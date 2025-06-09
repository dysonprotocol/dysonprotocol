document.addEventListener("alpine:init", () => {
  Alpine.data("messagesStore", () => ({
    messages: [],
    message: "",
    sponsorAmount: 0,
    deletingMessageId: null,

    init() {
      this.fetchMessages();
    },

    // Get the current script address from the domain
    getScriptAddress() {
      // Extract script address from subdomain (e.g., dys1prvefcdgdnh2cpas6rnnval84n0gv2r28tklr4.localhost:1317)
      const hostname = window.location.hostname;
      const parts = hostname.split('.');
      return parts[0]; // First part should be the script address
    },

    async fetchMessages() {
      try {
        const scriptAddress = this.getScriptAddress();
        const apiUrl = window.location.origin; // Use current domain
        const fetchUrl = `${apiUrl}/dysonprotocol/storage/v1/storage_list?owner=${scriptAddress}&index_prefix=messages&pagination.limit=100&pagination.reverse=true`;
        console.log("Fetching messages from:", fetchUrl);
        
        const resp = await fetch(fetchUrl);
        console.log("Fetch response status:", resp.status);
        
        const data = await resp.json();
        console.log("Raw fetch data:", data);
        
        if (data?.entries) {
          this.messages = data.entries.map((s) => ({
            ...s,
            data: JSON.parse(s.data),
          }));
          console.log("Parsed messages:", this.messages);
        } else {
          this.messages = [];
          console.log("No entries found in response");
        }
      } catch (error) {
        console.log("Error fetching messages:", error?.message || error);
        this.messages = [];
      }
    },

    async submitMessage() {
      try {
        console.log("Running script...");
        console.log("submitMessage called");
        
        const scriptAddress = this.getScriptAddress();
        console.log("Script address:", scriptAddress);
        
        const walletStore = Alpine.store("walletStore");
        console.log("Wallet store:", walletStore ? "found" : "not found");
        
        if (!walletStore) {
          console.log("Error: Wallet store not found");
          return;
        }
        
        if (!walletStore.runDysonScript) {
          console.log("Error: runDysonScript function not found");
          return;
        }

        // Prepare attached messages if sponsor amount is provided
        let attachedMsg = [];
        if (this.sponsorAmount && this.sponsorAmount > 0) {
          const bankSendMsg = {
            "@type": "/cosmos.bank.v1beta1.MsgSend",
            "from_address": walletStore.activeWalletMeta.address,
            "to_address": scriptAddress,
            "amount": [{
              "denom": "dys",
              "amount": this.sponsorAmount.toString()
            }]
          };
          attachedMsg.push(bankSendMsg);
          console.log("Adding bank send message:", bankSendMsg);
        }

        let finalGasLimit = 200000; // Default gas limit

        // If using CosmJS wallet, simulate first to get gas estimate
        if (walletStore.activeWalletMeta?.type === "cosmjs") {
          console.log("CosmJS wallet detected, simulating transaction first...");
          
          const simulationResult = await walletStore.runDysonScript({
            scriptAddress: scriptAddress,
            functionName: "save_message",
            kwargs: JSON.stringify({ message: this.message }),
            attachedMsg: attachedMsg,
            gasLimit: finalGasLimit,
            simulate: true
          });
          
          console.log("Simulation result:", simulationResult);
          
          if (simulationResult.success && simulationResult.rawSendMsgsResponse?.gasUsed) {
            const simulatedGas = parseInt(simulationResult.rawSendMsgsResponse.gasUsed, 10);
            // Add 50% buffer to the simulated gas usage
            finalGasLimit = Math.ceil(simulatedGas * 1.5);
            console.log(`Gas simulation: used ${simulatedGas}, setting limit to ${finalGasLimit}`);
          } else {
            console.log("Simulation failed or no gas info, using default gas limit");
            if (!simulationResult.success) {
              console.log("Simulation error:", simulationResult.rawSendMsgsResponse?.rawLog);
              return; // Don't proceed if simulation fails
            }
          }
        }
        
        console.log("Calling runDysonScript with:", JSON.stringify({
          scriptAddress: scriptAddress,
          functionName: "save_message", 
          kwargs: JSON.stringify({ message: this.message }),
          attachedMsg: attachedMsg,
          gasLimit: finalGasLimit,
        }));
        
        const result = await walletStore.runDysonScript({
          scriptAddress: scriptAddress,
          functionName: "save_message",
          kwargs: JSON.stringify({ message: this.message }),
          attachedMsg: attachedMsg,
          gasLimit: finalGasLimit,
          simulate: false
        });
        
        console.log("runDysonScript result:", JSON.stringify(result, null, 2));

        // Clear the message input and sponsor amount on successful submission
        if (result.success) {
          this.message = "";
          this.sponsorAmount = 0;
          console.log("Message and sponsor amount cleared after successful submission");
        }

        // Refresh the messages list
        await this.fetchMessages();

        // Optionally scroll to new message index
        const newMessageIndex = result?.scriptResponse?.result;
        if (newMessageIndex) {
          location.hash = newMessageIndex;
        }
      } catch (error) {
        console.log("submitMessage error:", error?.message || error);
        console.log("Script error:", error?.message || error);
      }
    },

    async deleteMessage(messageId) {
      try {
        console.log("Deleting message:", messageId);
        
        // Show confirmation dialog
        if (!confirm("Are you sure you want to delete this message? This action cannot be undone.")) {
          console.log("Delete cancelled by user");
          return;
        }
        
        // Set loading state
        this.deletingMessageId = messageId;
        
        const scriptAddress = this.getScriptAddress();
        const walletStore = Alpine.store("walletStore");
        
        if (!walletStore || !walletStore.runDysonScript) {
          console.log("Error: Wallet store or runDysonScript not found");
          return;
        }

        let finalGasLimit = 200000; // Default gas limit

        // If using CosmJS wallet, simulate first to get gas estimate
        if (walletStore.activeWalletMeta?.type === "cosmjs") {
          console.log("CosmJS wallet detected, simulating delete transaction first...");
          
          const simulationResult = await walletStore.runDysonScript({
            scriptAddress: scriptAddress,
            functionName: "delete_message",
            kwargs: JSON.stringify({ message_id: messageId }),
            gasLimit: finalGasLimit,
            simulate: true
          });
          
          console.log("Delete simulation result:", simulationResult);
          
          if (simulationResult.success && simulationResult.rawSendMsgsResponse?.gasUsed) {
            const simulatedGas = parseInt(simulationResult.rawSendMsgsResponse.gasUsed, 10);
            finalGasLimit = Math.ceil(simulatedGas * 1.5);
            console.log(`Delete gas simulation: used ${simulatedGas}, setting limit to ${finalGasLimit}`);
          } else {
            console.log("Delete simulation failed or no gas info, using default gas limit");
            if (!simulationResult.success) {
              console.log("Delete simulation error:", simulationResult.rawSendMsgsResponse?.rawLog);
              this.deletingMessageId = null;
              return;
            }
          }
        }
        
        console.log("Calling delete_message with:", JSON.stringify({
          scriptAddress: scriptAddress,
          functionName: "delete_message", 
          kwargs: JSON.stringify({ message_id: messageId }),
          gasLimit: finalGasLimit,
        }));
        
        const result = await walletStore.runDysonScript({
          scriptAddress: scriptAddress,
          functionName: "delete_message",
          kwargs: JSON.stringify({ message_id: messageId }),
          gasLimit: finalGasLimit,
          simulate: false
        });
        
        console.log("deleteMessage result:", JSON.stringify(result, null, 2));

        if (result.success) {
          console.log("Message deleted successfully");
          // Refresh the messages list
          await this.fetchMessages();
        } else {
          console.log("Delete failed:", result);
          // You could add user-friendly error handling here
        }

      } catch (error) {
        console.log("deleteMessage error:", error?.message || error);
      } finally {
        // Clear loading state
        this.deletingMessageId = null;
      }
    },
  }));
}); 