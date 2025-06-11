document.addEventListener("alpine:init", () => {
  Alpine.data("storageStore", () => ({
    // Storage entries
    entries: [],
    filteredEntries: [],
    loading: false,
    searchTerm: "",
    indexPrefix: "",
    newEntryIndex: "",
    
    // Selected entry for content viewing
    selectedEntry: null,
    originalContent: '', // Track original content to detect changes
    hasContentChanged: false,
    
    // Pagination
    currentPage: 1,
    pageSize: 20,
    totalEntries: 0,
    hasMore: false,
    offset: 0, // Track the current offset for pagination
    
    // Can delete flag
    canDelete: false,

    init() {
      // Ensure filteredEntries is always an array
      this.filteredEntries = [];
      
      this.loadEntries();
      this.checkCanDelete();
      // Watch for search changes
      this.$watch('searchTerm', () => {
        this.filterEntries();
      });
      // Watch for wallet changes
      this.$watch('$store.walletStore.activeWalletMeta', () => {
        this.checkCanDelete();
      });
    },
    
    // Get the current script address from the domain
    getScriptAddress() {
      const hostname = window.location.hostname;
      const parts = hostname.split('.');
      return parts[0];
    },

    async loadEntries(append = false) {
      try {
        if (!append) {
          this.loading = true;
          this.currentPage = 1;
          this.offset = 0;
        }
        
        const scriptAddress = this.getScriptAddress();
        const apiUrl = window.location.origin;
        
        // Build query parameters
        const params = new URLSearchParams({
          owner: scriptAddress,
          index_prefix: this.indexPrefix,
          'pagination.limit': this.pageSize.toString(),
          'pagination.offset': this.offset.toString(),
          'pagination.reverse': 'false'
        });
        
        // No need for pagination key - using offset
        
        const fetchUrl = `${apiUrl}/dysonprotocol/storage/v1/storage_list?${params}`;
        console.log("Fetching storage entries from:", fetchUrl);
        
        const resp = await fetch(fetchUrl);
        const data = await resp.json();
        
        if (data?.entries) {
          const newEntries = data.entries.map(entry => ({
            ...entry,
            dataSize: new Blob([entry.data]).size,
            dataPreview: this.generatePreview(entry.data)
          }));
          
          if (append) {
            // Filter out duplicates based on index
            const existingIndexes = new Set(this.entries.map(e => e.index));
            const uniqueNewEntries = newEntries.filter(entry => !existingIndexes.has(entry.index));
            this.entries = [...this.entries, ...uniqueNewEntries];
          } else {
            this.entries = newEntries;
            // Clear selection if we're reloading
            this.selectedEntry = null;
          }
          
          // Update pagination state
          this.hasMore = newEntries.length === this.pageSize;
          
          this.filterEntries();
        } else {
          if (!append) {
            this.entries = [];
            this.filteredEntries = [];
            this.selectedEntry = null;
          }
          this.hasMore = false;
        }
      } catch (error) {
        console.error("Error loading storage entries:", error);
        if (!append) {
          this.entries = [];
          this.filteredEntries = [];
          this.selectedEntry = null;
        }
        this.hasMore = false;
      } finally {
        this.loading = false;
      }
    },
    
    filterEntries() {
      // Ensure entries is always an array
      const safeEntries = Array.isArray(this.entries) ? this.entries : [];
      
      // Only apply search term filter in memory (prefix is handled by API)
      if (!this.searchTerm.trim()) {
        this.filteredEntries = [...safeEntries];
      } else {
        const searchLower = this.searchTerm.toLowerCase();
        this.filteredEntries = safeEntries.filter(entry =>
          entry.index.toLowerCase().includes(searchLower) ||
          entry.data.toLowerCase().includes(searchLower)
        );
      }
      
      // Clear selection if the selected entry is no longer in filtered results
      if (this.selectedEntry && !this.filteredEntries.find(entry => entry.index === this.selectedEntry.index)) {
        this.selectedEntry = null;
      }
    },
    
    generatePreview(data, maxLength = 100) {
      if (data.length <= maxLength) {
        return data;
      }
      return data.substring(0, maxLength) + '...';
    },
    
    formatSize(bytes) {
      if (bytes === 0) return '0 B';
      const k = 1024;
      const sizes = ['B', 'KB', 'MB', 'GB'];
      const i = Math.floor(Math.log(bytes) / Math.log(k));
      return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
    },
    
    // Entry selection
    selectEntry(entry) {
      this.selectedEntry = entry;
      this.originalContent = entry ? entry.data : '';
      this.hasContentChanged = false;
    },
    
    // Track content changes
    onContentChange(newContent) {
      this.hasContentChanged = newContent !== this.originalContent;
    },

    // Check if current wallet can delete entries
    checkCanDelete() {
      console.log('=== Checking if can delete ===');
      
      // Get script address from hostname
      const scriptAddress = this.getScriptAddress();
      console.log('Script address from hostname:', scriptAddress);
      
      // Check if wallet store exists
      const walletStore = Alpine.store('walletStore');
      console.log('Wallet store exists:', !!walletStore);
      
      if (!walletStore) {
        this.canDelete = false;
        console.log('No wallet store - canDelete = false');
        return;
      }
      
      // Check if wallet is connected
      console.log('Active wallet meta:', walletStore.activeWalletMeta);
      
      if (!walletStore.activeWalletMeta) {
        this.canDelete = false;
        console.log('No active wallet - canDelete = false');
        return;
      }
      
             // Get wallet address
       const walletAddress = walletStore.activeWalletMeta.address;
       console.log('Wallet address:', walletAddress);
      
      // Compare addresses
      const matches = walletAddress === scriptAddress;
      console.log('Addresses match:', matches);
      
      this.canDelete = matches;
      console.log('Final canDelete value:', this.canDelete);
    },
    
    // Copy content to clipboard
    async copyContent() {
      if (!this.selectedEntry) return;
      
      try {
        await navigator.clipboard.writeText(this.selectedEntry.data);
        // You could add a toast notification here
        console.log('Content copied to clipboard');
      } catch (error) {
        console.error('Failed to copy content:', error);
        // Fallback for older browsers
        const textArea = document.createElement('textarea');
        textArea.value = this.selectedEntry.data;
        document.body.appendChild(textArea);
        textArea.select();
        document.execCommand('copy');
        document.body.removeChild(textArea);
      }
    },
    
    async deleteEntry(index) {
      
      try {
        const walletStore = Alpine.store('walletStore');
        if (!walletStore) {
          throw new Error('Wallet store not available');
        }
        
        const scriptAddress = this.getScriptAddress();
        
        const msg = {
          "@type": "/dysonprotocol.storage.v1.MsgStorageDelete",
          "owner": scriptAddress,
          "indexes": [index]
        };
        
        const result = await walletStore.sendMsg({
          msg: msg,
        });
        
        if (result.success) {
          // Clear selection if we're deleting the selected entry
          if (this.selectedEntry && this.selectedEntry.index === index) {
            this.selectedEntry = null;
          }
          await this.loadEntries();
        } else {
          console.error("Delete failed:", result);
        }
        
      } catch (error) {
        console.error("Error deleting entry:", error);
      }
    },
    
    // Utility methods
    loadMore() {
      this.offset += this.pageSize;
      this.loadEntries(true);
    },
    

    
    // Update storage entry
    async updateStorage() {
      if (!this.selectedEntry || !this.hasContentChanged) return;
      
      try {
        const walletStore = Alpine.store('walletStore');
        if (!walletStore) {
          throw new Error('Wallet store not available');
        }
        
        const scriptAddress = this.getScriptAddress();
        // Get the current content from the pre element
        const preElement = document.querySelector('pre[contenteditable]');
        const newContent = preElement ? preElement.textContent : this.selectedEntry.data;
        
        const msg = {
          "@type": "/dysonprotocol.storage.v1.MsgStorageSet",
          "owner": scriptAddress,
          "index": this.selectedEntry.index,
          "data": newContent
        };
        
        const result = await walletStore.sendMsg({
          msg: msg,
        });
        
        if (result.success) {
          // Update the original content and reset change flag
          this.originalContent = newContent;
          this.hasContentChanged = false;
          
          // Refresh the entry in our list
          const entryIndex = this.entries.findIndex(e => e.index === this.selectedEntry.index);
          if (entryIndex !== -1) {
            this.entries[entryIndex] = {
              ...this.entries[entryIndex],
              data: newContent,
              dataSize: new Blob([newContent]).size,
              dataPreview: this.generatePreview(newContent)
            };
            this.selectedEntry = this.entries[entryIndex];
            this.filterEntries();
          }
          
          console.log('Storage updated successfully');
        } else {
          console.error("Storage update failed:", result);
        }
        
      } catch (error) {
        console.error("Error updating storage:", error);
      }
    },
    
    // Create new storage entry
    createNewEntry() {
      if (!this.newEntryIndex.trim() || !this.canDelete) return;
      
      // Create a new entry object
      const newEntry = {
        index: this.newEntryIndex.trim(),
        data: '',
        dataSize: 0,
        dataPreview: ''
      };
      
      // Add to entries list (at the beginning for visibility)
      this.entries.unshift(newEntry);
      this.filterEntries();
      
      // Select the new entry
      this.selectEntry(newEntry);
      
      // Clear the input
      this.newEntryIndex = '';
      
      // Set flag to indicate this is a new entry that needs saving
      this.hasContentChanged = true;
      
      console.log('Created new entry:', newEntry.index);
    }
  }));
}); 