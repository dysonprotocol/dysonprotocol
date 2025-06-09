// dysonTxUtils.js
import { Tx, TxBody, AuthInfo, TxRaw } from "cosmjs-types/cosmos/tx/v1beta1/tx.js";
import { fromBase64, toBase64 } from "@cosmjs/encoding";

import { DirectSecp256k1HdWallet, makeSignDoc } from "@cosmjs/proto-signing";
import { Any } from "cosmjs-types/google/protobuf/any.js";
import { SignMode } from "cosmjs-types/cosmos/tx/signing/v1beta1/signing.js";

/** Fetch chain info for address. */
export async function getChainInfo({ apiUrl, address }) {
  const nodeInfoResp = await fetch(`${apiUrl}/cosmos/base/tendermint/v1beta1/node_info`);
  if (!nodeInfoResp.ok) {
    throw new Error(`Failed to fetch node info: ${await nodeInfoResp.text()}`);
  }
  const nodeInfoJson = await nodeInfoResp.json();
  const chainId = nodeInfoJson.default_node_info.network;

  const acctInfoResp = await fetch(`${apiUrl}/cosmos/auth/v1beta1/account_info/${address}`);
  if (!acctInfoResp.ok) {
    // try to parse as json: {"code":5,"message":"account dys123123 not found","details":[]}
    const textErr = await acctInfoResp.text();
    let jsonErr;
    try {
      jsonErr = JSON.parse(textErr);
      if (jsonErr.code === 5) {
        return { chainId, accountNumber: 0, sequence: 0 };
      }
    } catch {
      // ignore parse error
    }
    throw new Error(`Failed to fetch account info: ${textErr}`);
  }
  const acctInfoJson = await acctInfoResp.json();
  const accountNumber = parseInt(acctInfoJson.info.account_number, 10);
  const sequence = parseInt(acctInfoJson.info.sequence, 10);

  return { chainId, accountNumber, sequence };
}

/** Build a base Tx object. */
export function prepareTx({ msgs, memo = "", fee }) {
  return {
    body: {
      messages: msgs,
      memo,
      timeout_height: "0",
      unordered: false,
      timeout_timestamp: "0001-01-01T00:00:00Z",
      extension_options: [],
      non_critical_extension_options: [],
    },
    auth_info: {
      signer_infos: [],
      fee: fee || {
        amount: [],
        gas_limit: "200000",
      },
      tip: null,
    },
    signatures: [],
  };
}

/** Insert signer info so chain sees exactly 1 signer. */
export function addSignerInfo({ transaction, pubkey, sequence }) {
  transaction.auth_info.signer_infos = [
    {
      public_key: {
        "@type": "/cosmos.crypto.secp256k1.PubKey",
        key: toBase64(pubkey),
      },
      mode_info: { single: { mode: "SIGN_MODE_DIRECT" } }, // SIGN_MODE_DIRECT
      sequence: String(sequence),
    },
  ];
  return transaction;
}

/**
 * Encode the transaction via /cosmos/tx/v1beta1/encode => returns base64.
 * Then decode to get bodyBytes/authInfoBytes.
 */
export async function encodeAndDecodeTx({ apiUrl, transaction }) {
  const encodeRes = await fetch(`${apiUrl}/cosmos/tx/v1beta1/encode`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ tx: transaction }),
  });
  if (!encodeRes.ok) {
    throw new Error(`Failed to encode transaction: ${await encodeRes.text()}`);
  }
  const { tx_bytes: txBytesBase64 } = await encodeRes.json();
  const protoTx = Tx.decode(fromBase64(txBytesBase64));
  const bodyBytes = TxBody.encode(protoTx.body).finish();
  const authInfoBytes = AuthInfo.encode(protoTx.authInfo).finish();
  return { txBytesBase64, bodyBytes, authInfoBytes };
}

/** Sign the Tx using CosmJS or Leap/Keplr signDirect. Returns raw bytes. */
export async function signTx({
  wallet,
  walletType,
  chainId,
  address,
  accountNumber,
  sequence,
  bodyBytes,
  authInfoBytes,
}) {
  const signDoc = {
    bodyBytes,
    authInfoBytes,
    chainId,
    accountNumber: Number(accountNumber),
  };

  let directSignResponse;
  directSignResponse = await wallet.signDirect(address, {
    bodyBytes,
    authInfoBytes,
    chainId,
    accountNumber,
    sequence,
  });

  const { signed, signature } = directSignResponse;
  const txRaw = TxRaw.fromPartial({
    bodyBytes: signed.bodyBytes || bodyBytes,
    authInfoBytes: signed.authInfoBytes || authInfoBytes,
    signatures: [fromBase64(signature.signature)],
  });
  return TxRaw.encode(txRaw).finish();
}

/** Strips trailing ": script execution error" if present. */
function stripScriptSuffix(log) {
  const suffix = ": script execution error";
  return log.endsWith(suffix) ? log.slice(0, -suffix.length).trim() : log;
}

/** Validate msg types appear in events => type="message" => attr.key="action". */
function checkMissingMsgTypes(events, msgTypes) {
  if (!Array.isArray(events)) return msgTypes;
  const messageEvents = events.filter((e) => e.type === "message");
  const foundActions = new Set();
  for (const me of messageEvents) {
    for (const attr of me.attributes || []) {
      if (attr.key === "action" && attr.value) {
        foundActions.add(attr.value);
      }
    }
  }
  return msgTypes.filter((t) => !foundActions.has(t));
}

/**
 * Helper that fetches either /simulate or /txs, optionally polls if broadcast,
 * then returns a unified result shape.
 */
async function submitTx({ apiUrl, txRawBytesBase64, msgTypes, mode }) {
  if (mode === "simulate") {
    // Single POST => /simulate
    const simRes = await fetch(`${apiUrl}/cosmos/tx/v1beta1/simulate`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ tx_bytes: txRawBytesBase64 }),
    });
    if (!simRes.ok) {
      const textErr = await simRes.text();
      try {
        const jsonErr = JSON.parse(textErr);
        return {
          kind: "simulate",
          success: false,
          code: 1,
          gasUsed: "0",
          rawLog: jsonErr?.message || textErr,
          raw: jsonErr,
        };
      } catch {
        // ignore parse error
      }

      return {
        kind: "simulate",
        success: false,
        code: 1,
        gasUsed: "0",
        rawLog: "",
        raw: null,
      };
    }
    const simData = await simRes.json();
    const gasUsed = simData?.gas_info?.gas_used || "0";

    let code = 0;
    let rawLog = "";
    const events = simData?.result?.events || [];

    // check missing message types
    const missingTypes = checkMissingMsgTypes(events, msgTypes);
    if (missingTypes.length > 0) {
      code = 1;
      rawLog = "";
      return {
        kind: "simulate",
        success: false,
        code,
        gasUsed,
        rawLog,
        raw: simData,
      };
    }
    return {
      kind: "simulate",
      success: true,
      code,
      gasUsed,
      rawLog,
      raw: simData,
    };
  }

  // mode === "broadcast"
  // 1) broadcast
  const broadcastRes = await fetch(`${apiUrl}/cosmos/tx/v1beta1/txs`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      tx_bytes: txRawBytesBase64,
      mode: "BROADCAST_MODE_SYNC",
    }),
  });
  if (!broadcastRes.ok) {
    const textErr = await broadcastRes.text();
    return {
      kind: "broadcast",
      success: false,
      code: 1,
      gasUsed: "0",
      rawLog: `Broadcast error: ${textErr}`,
      raw: null,
    };
  }
  const broadcastData = await broadcastRes.json();
  const txHash = broadcastData?.tx_response?.txhash;
  if (!txHash) {
    return {
      kind: "broadcast",
      success: false,
      code: 1,
      gasUsed: "0",
      rawLog: "No txhash in broadcast response",
      raw: broadcastData,
    };
  }

  // Check if broadcast failed immediately (code != 0)
  const broadcastCode = broadcastData?.tx_response?.code || 0;
  if (broadcastCode !== 0) {
    const broadcastLog = broadcastData?.tx_response?.raw_log || "";
    const broadcastGasUsed = broadcastData?.tx_response?.gas_used || "0";
    return {
      kind: "broadcast",
      success: false,
      code: broadcastCode,
      gasUsed: broadcastGasUsed,
      rawLog: broadcastLog,
      raw: broadcastData,
    };
  }

  // 2) poll for final
  const maxAttempts = 10;
  const intervalMs = 1000;
  let finalData = null;
  for (let i = 0; i < maxAttempts; i++) {
    const res = await fetch(`${apiUrl}/cosmos/tx/v1beta1/txs/${txHash}`);
    if (res.ok) {
      finalData = await res.json();
      break;
    }
    await new Promise((resolve) => setTimeout(resolve, intervalMs));
  }
  if (!finalData) {
    return {
      kind: "broadcast",
      success: false,
      code: 1,
      gasUsed: "0",
      rawLog: `Transaction not found after ${maxAttempts} attempts: ${txHash}`,
      raw: null,
    };
  }

  // 3) parse final
  const txResp = finalData?.tx_response;
  if (!txResp) {
    return {
      kind: "broadcast",
      success: false,
      code: 1,
      gasUsed: "0",
      rawLog: "No tx_response in final data",
      raw: finalData,
    };
  }
  const code = txResp.code || 0;
  const gasUsed = txResp.gas_used || "0";
  const rawOriginalLog = txResp.raw_log || "";
  const strippedLog = stripScriptSuffix(rawOriginalLog);

  // code != 0 => failure
  if (code !== 0) {
    return { kind: "broadcast", success: false, code, gasUsed, rawLog: rawOriginalLog, raw: finalData };
  }

  // code=0 => success => check missing msg types
  const events = txResp.events || [];
  const missingTypes = checkMissingMsgTypes(events, msgTypes);
  if (missingTypes.length > 0) {
    return {
      kind: "broadcast",
      success: false,
      code,
      gasUsed,
      rawLog: rawOriginalLog,
      raw: finalData,
    };
  }
  return {
    kind: "broadcast",
    success: true,
    code,
    gasUsed,
    rawLog: strippedLog,
    raw: finalData,
  };
}

/**
 * High-level function:
 * 1) getChainInfo
 * 2) prepare Tx (with signer info)
 * 3) encode & decode => sign => get final raw bytes
 * 4) optionally simulate or broadcast => returns { kind, success, code, gasUsed, rawLog, raw }
 */
export async function sendMsgs({ apiUrl, wallet, walletType, address, msgs, memo = "", fee, simulate = false }) {
  const { chainId, accountNumber, sequence } = await getChainInfo({ apiUrl, address });

  let transaction = prepareTx({ msgs, memo, fee });
  const [{ pubkey }] = await wallet.getAccounts();
  transaction = addSignerInfo({ transaction, pubkey, sequence });

  // We'll capture each message's @type
  const msgTypes = msgs.map((m) => m["@type"] || "");

  const { txBytesBase64, bodyBytes, authInfoBytes } = await encodeAndDecodeTx({ apiUrl, transaction });
  const signedTxRawBytes = await signTx({
    wallet,
    walletType,
    chainId,
    address,
    accountNumber,
    sequence,
    bodyBytes,
    authInfoBytes,
  });
  const signedTxRawB64 = toBase64(signedTxRawBytes);

  const mode = simulate ? "simulate" : "broadcast";
  return submitTx({ apiUrl, txRawBytesBase64: signedTxRawB64, msgTypes, mode });
}

/**
 * runScript => convenience wrapper for Dyson scripts => calls sendMsgs
 * but also parses scriptResponse differently for success/fail.
 */
export async function runScript({
  apiUrl,
  wallet,
  walletType,
  executorAddress,
  scriptAddress,
  functionName = "",
  args = "",
  kwargs = "",
  extraCode = "",
  attachedMsg = [],
  memo = "",
  fee,
  simulate = false,
}) {
  const msg = {
    "@type": "/dysonprotocol.script.v1.MsgExec",
    executor_address: executorAddress,
    script_address: scriptAddress,
    function_name: functionName,
    args,
    kwargs,
    extra_code: extraCode,
    attached_messages: attachedMsg,
  };

  if (scriptAddress === "") {
    throw new Error("scriptAddress is required");
  }

  const sendResult = await sendMsgs({
    apiUrl,
    wallet,
    walletType,
    address: executorAddress,
    msgs: [msg],
    memo,
    fee,
    simulate,
  });

  const { kind, success, code, rawLog, raw } = sendResult;
  let scriptResponse = null;

  if (success) {
    // parse from events
    const events = kind === "simulate" ? raw?.result?.events : raw?.tx_response?.events;
    scriptResponse = parseScriptResponse(events);
  } else {
    // parse from rawLog if code != 0
    try {
      // Find the first { and last } to extract JSON from any error format
      const firstBrace = rawLog.indexOf('{');
      const lastBrace = rawLog.lastIndexOf('}');
      
      if (firstBrace !== -1 && lastBrace !== -1 && lastBrace > firstBrace) {
        // Extract JSON between first { and last }
        const jsonStr = rawLog.substring(firstBrace, lastBrace + 1);
        scriptResponse = JSON.parse(jsonStr);
      } else {
        // Fallback: try removing trailing ": script execution error"
        const cleanedLog = rawLog.replace(/(: script execution error)$/, "");
        scriptResponse = JSON.parse(cleanedLog);
      }
    } catch {
      scriptResponse = null;
    }
  }

  return {
    kind,
    success,
    scriptResponse,
    rawSendMsgsResponse: sendResult,
  };
}

function parseScriptResponse(events) {
  if (!Array.isArray(events)) return null;
  const scriptEvt = events.find((e) => e.type === "dysonprotocol.script.v1.EventExecScript");
  if (!scriptEvt?.attributes) return null;

  const responseAttr = scriptEvt.attributes.find((a) => a.key === "response");
  if (!responseAttr?.value) return null;

  try {
    const parsed = JSON.parse(responseAttr.value.replace(/(: script execution error)$/, ""));
    if (parsed.result && typeof parsed.result === "string") {
      try {
        parsed.result = JSON.parse(parsed.result);
      } catch {
        // ignore parse error
      }
    }
    return parsed.result;
  } catch {
    return responseAttr.value; // fallback
  }
}

/**
 * signData => create a single "/offchain.MsgSignArbitraryData" Tx, sign it, return the signed Tx raw bytes (base64).
 * This DOES NOT broadcast or simulate. Just signs.
 */
export async function signData({ apiUrl, wallet, walletType, address, data }) {
  const { accountNumber } = await getChainInfo({ apiUrl, address });
  const sequence = "0";
  const chainId = "";

  // Create the /offchain.MsgSignArbitraryData message
  const msg = {
    "@type": "/offchain.MsgSignArbitraryData",
    signer: address,
    app_domain: "dysond",
    data: data, // Provide as string or base64 if needed
  };
  const fee = {
    amount: [],
    gas_limit: "0",
    payer: "",
    granter: "",
  };
  const memo = "";
  // Build the Tx with one message
  let transaction = prepareTx({ msgs: [msg], memo, fee });
  console.log("prepareTx transaction", transaction);
  const [{ pubkey }] = await wallet.getAccounts();
  transaction = addSignerInfo({ transaction, pubkey, sequence });
  console.log("addSignerInfo transaction", transaction);
  console.log("Pubkey", pubkey);

  // Encode/decode
  const { bodyBytes, authInfoBytes } = await encodeAndDecodeTx({ apiUrl, transaction });

  // Sign
  const signedTxRawBytes = await signTx({
    wallet,
    walletType,
    chainId,
    address,
    accountNumber,
    sequence,
    bodyBytes,
    authInfoBytes,
  });

  // Return as base64 (or raw Uint8Array if you prefer)
  /*
   * const tx = TxRaw.decode(signedTxRawBytes);
    return {
    address,
    pubkey: toBase64(pubkey),
  };
  */
  //transaction.signatures = decodedTxRaw.signatures.map(toBase64);
  const decodedTxRaw = decodeTxRaw(signedTxRawBytes);
  return {
    authInfo: decodedTxRaw.authInfo,
    body: decodedTxRaw.body,
    signatures: decodedTxRaw.signatures,
  };
}

export function decodeTxRaw(tx) {
  const txRaw = TxRaw.decode(tx);
  return {
    authInfo: AuthInfo.decode(txRaw.authInfoBytes),
    body: TxBody.decode(txRaw.bodyBytes),
    signatures: txRaw.signatures.map(toBase64),
  };
} 