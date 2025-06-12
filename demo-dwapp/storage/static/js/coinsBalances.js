// coinsBalances.js
// Registers balancesComponent for /coins page

document.addEventListener('alpine:init', () => {
  Alpine.data('balancesComponent', () => ({
    balances: [],
    loading: false,
    error: '',
    get walletConnected() {
      return !!$store.walletStore.activeWalletMeta;
    },
    async loadBalances() {
      if (!this.walletConnected) {
        this.balances = [];
        return;
      }
      this.loading = true;
      try {
        const apiUrl = $store.walletStore.restUrl;
        const address = $store.walletStore.activeWalletMeta.address;
        const resp = await fetch(`${apiUrl}/cosmos/bank/v1beta1/balances/${address}`);
        if (!resp.ok) throw new Error(await resp.text());
        const data = await resp.json();
        const rawBalances = data?.balances || [];

        const results = [];
        for (const bal of rawBalances) {
          const denom = bal.denom;
          let symbol = denom;
          let exponent = 0;
          // best-effort metadata
          try {
            const metaRes = await fetch(`${apiUrl}/cosmos/bank/v1beta1/denoms_metadata/${denom}`);
            if (metaRes.ok) {
              const metaJson = await metaRes.json();
              const md = metaJson?.metadata || metaJson?.metadatas?.[0];
              if (md) {
                symbol = md.symbol || denom;
                const unit = (md.denom_units || []).find(u => u.denom === md.display);
                if (unit && unit.exponent != null) exponent = unit.exponent;
              }
            }
          } catch {}

          const rawAmount = bal.amount;
          let displayAmt = rawAmount;
          if (exponent > 0) {
            displayAmt = (Number(rawAmount) / Math.pow(10, exponent)).toLocaleString();
          }
          results.push({ denom, symbol, amount: displayAmt });
        }
        this.balances = results;
      } catch (e) {
        console.error(e);
        this.error = e.message;
      } finally {
        this.loading = false;
      }
    },
    init() {
      this.loadBalances();
      this.$watch('$store.walletStore.activeWalletMeta', () => this.loadBalances());
    }
  }));
}); 