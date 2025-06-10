document.addEventListener("alpine:init", () => {
  Alpine.data("tasksStore", () => ({
    // Module parameters
    moduleParams: {},
    moduleParamsLoading: false,
    
    // Tasks data
    tasks: [],
    filteredTasks: [],
    loading: false,
    
    // Pagination
    currentPage: 1,
    pageSize: 20,
    hasMore: false,
    totalTasks: 0,
    
    // Filters and sorting
    statusFilters: Alpine.$persist(["PENDING", "DONE", "FAILED", "EXPIRED"]),
    selectedStatuses: Alpine.$persist(["PENDING", "DONE"]),
    sortBy: Alpine.$persist("scheduled_timestamp"),
    sortOrder: Alpine.$persist("asc"),
    searchTerm: "",
    
    // Task creation
    isCreatingTask: false,
    createTaskLoading: false,
    newTask: {
      scheduledTimestamp: "",
      expiryTimestamp: "",
      gasLimit: 200000,
      gasFee: "0",
      messages: []
    },
    
    // Message builder
    isAddingMessage: false,
    currentMessageType: "bank_send",
    newMessage: {
      "@type": "/cosmos.bank.v1beta1.MsgSend",
      "from_address": "",
      "to_address": "",
      "amount": [{"denom": "dys", "amount": "1000"}]
    },
    newMessageJson: "",
    
    // Task templates
    taskTemplates: Alpine.$persist([
      {
        name: "Token Transfer",
        description: "Schedule a token transfer",
        messages: [{
          "@type": "/cosmos.bank.v1beta1.MsgSend",
          "from_address": "",
          "to_address": "",
          "amount": [{"denom": "dys", "amount": "1000"}]
        }]
      },
      {
        name: "Script Execution",
        description: "Schedule a script function call",
        messages: [{
          "@type": "/dysonprotocol.script.v1.MsgExec",
          "executor_address": "",
          "script_address": "",
          "function_name": "",
          "args": "[]",
          "kwargs": "{}",
          "extra_code": "",
          "attached_messages": []
        }]
      }
    ]),
    
    // Task details
    selectedTask: null,
    showTaskDetails: false,
    
    // Statistics
    stats: {
      total: 0,
      pending: 0,
      done: 0,
      failed: 0,
      expired: 0
    },

    init() {
      this.loadModuleParams();
      this.loadTasks();
      
      // Watch for filter changes
      this.$watch('selectedStatuses', () => {
        this.filterTasks();
      });
      this.$watch('searchTerm', () => {
        this.filterTasks();
      });
      this.$watch('sortBy', () => {
        this.sortTasks();
      });
      this.$watch('sortOrder', () => {
        this.sortTasks();
      });
    },
    
    // Get the current script address from the domain
    getScriptAddress() {
      const hostname = window.location.hostname;
      const parts = hostname.split('.');
      return parts[0];
    },

    // Load module parameters
    async loadModuleParams() {
      try {
        this.moduleParamsLoading = true;
        const apiUrl = window.location.origin;
        const response = await fetch(`${apiUrl}/dysonprotocol/crontask/v1/params`);
        
        if (response.ok) {
          const data = await response.json();
          this.moduleParams = data.params || {};
        } else {
          console.error("Failed to load module parameters:", response.statusText);
        }
      } catch (error) {
        console.error("Error loading module parameters:", error);
      } finally {
        this.moduleParamsLoading = false;
      }
    },

    // Load tasks
    async loadTasks(append = false) {
      try {
        if (!append) {
          this.loading = true;
          this.currentPage = 1;
        }
        
        const scriptAddress = this.getScriptAddress();
        const apiUrl = window.location.origin;
        
        // Build query parameters
        const params = new URLSearchParams({
          'pagination.limit': this.pageSize.toString(),
          'pagination.offset': ((this.currentPage - 1) * this.pageSize).toString()
        });
        
        const fetchUrl = `${apiUrl}/dysonprotocol/crontask/v1/tasks/creator/${scriptAddress}?${params}`;
        console.log("Fetching tasks from:", fetchUrl);
        
        const response = await fetch(fetchUrl);
        const data = await response.json();
        
        if (data?.tasks) {
          const newTasks = data.tasks.map(task => ({
            ...task,
            // Parse timestamps
            scheduledTime: new Date(parseInt(task.scheduled_timestamp) * 1000),
            expiryTime: task.expiry_timestamp ? new Date(parseInt(task.expiry_timestamp) * 1000) : null,
            // Parse messages
            parsedMessages: this.parseMessages(task.messages),
            // Calculate time remaining for pending tasks
            timeRemaining: this.calculateTimeRemaining(task)
          }));
          
          if (append) {
            this.tasks = [...this.tasks, ...newTasks];
          } else {
            this.tasks = newTasks;
          }
          
          this.hasMore = newTasks.length === this.pageSize;
          this.updateStats();
          this.filterTasks();
        } else {
          if (!append) {
            this.tasks = [];
            this.filteredTasks = [];
          }
        }
      } catch (error) {
        console.error("Error loading tasks:", error);
        this.tasks = [];
        this.filteredTasks = [];
      } finally {
        this.loading = false;
      }
    },

    // Parse messages for display
    parseMessages(messages) {
      return messages.map(msg => {
        try {
          const parsed = typeof msg === 'string' ? JSON.parse(msg) : msg;
          return {
            ...parsed,
            displayType: this.getMessageDisplayType(parsed["@type"]),
            summary: this.getMessageSummary(parsed)
          };
        } catch (error) {
          return {
            raw: msg,
            displayType: "Unknown",
            summary: "Invalid message format"
          };
        }
      });
    },

    // Get human-readable message type
    getMessageDisplayType(type) {
      const typeMap = {
        "/cosmos.bank.v1beta1.MsgSend": "Token Transfer",
        "/dysonprotocol.script.v1.MsgExec": "Script Execution",
        "/dysonprotocol.storage.v1.MsgStorageSet": "Storage Set",
        "/dysonprotocol.storage.v1.MsgStorageDelete": "Storage Delete",
        "/dysonprotocol.crontask.v1.MsgCreateTask": "Create Task",
        "/dysonprotocol.crontask.v1.MsgDeleteTask": "Delete Task"
      };
      return typeMap[type] || type.replace(/^.*\./, "");
    },

    // Get message summary for display
    getMessageSummary(message) {
      const type = message["@type"];
      
      switch (type) {
        case "/cosmos.bank.v1beta1.MsgSend":
          const amount = message.amount?.[0];
          return `Send ${amount?.amount || "?"} ${amount?.denom || "?"} to ${this.truncateAddress(message.to_address)}`;
        
        case "/dysonprotocol.script.v1.MsgExec":
          return `Call ${message.function_name || "function"}() on ${this.truncateAddress(message.script_address)}`;
        
        case "/dysonprotocol.storage.v1.MsgStorageSet":
          return `Set storage key: ${this.truncateString(message.index, 30)}`;
        
        case "/dysonprotocol.storage.v1.MsgStorageDelete":
          const indexes = message.indexes || [];
          return `Delete ${indexes.length} storage key(s)`;
        
        default:
          return `${this.getMessageDisplayType(type)} message`;
      }
    },

    // Calculate time remaining for pending tasks
    calculateTimeRemaining(task) {
      if (task.status !== "PENDING") return null;
      
      const now = new Date();
      const scheduled = new Date(parseInt(task.scheduled_timestamp) * 1000);
      const diff = scheduled.getTime() - now.getTime();
      
      if (diff <= 0) return "Overdue";
      
      const days = Math.floor(diff / (24 * 60 * 60 * 1000));
      const hours = Math.floor((diff % (24 * 60 * 60 * 1000)) / (60 * 60 * 1000));
      const minutes = Math.floor((diff % (60 * 60 * 1000)) / (60 * 1000));
      
      if (days > 0) return `${days}d ${hours}h`;
      if (hours > 0) return `${hours}h ${minutes}m`;
      return `${minutes}m`;
    },

    // Update statistics
    updateStats() {
      this.stats = {
        total: this.tasks.length,
        pending: this.tasks.filter(t => t.status === "PENDING").length,
        done: this.tasks.filter(t => t.status === "DONE").length,
        failed: this.tasks.filter(t => t.status === "FAILED").length,
        expired: this.tasks.filter(t => t.status === "EXPIRED").length
      };
    },

    // Filter tasks
    filterTasks() {
      let filtered = [...this.tasks];
      
      // Filter by status
      if (this.selectedStatuses.length > 0) {
        filtered = filtered.filter(task => this.selectedStatuses.includes(task.status));
      }
      
      // Filter by search term
      if (this.searchTerm.trim()) {
        const searchLower = this.searchTerm.toLowerCase();
        filtered = filtered.filter(task => 
          task.task_id.toLowerCase().includes(searchLower) ||
          task.status.toLowerCase().includes(searchLower) ||
          task.parsedMessages.some(msg => 
            msg.summary.toLowerCase().includes(searchLower)
          )
        );
      }
      
      this.filteredTasks = filtered;
      this.sortTasks();
    },

    // Sort tasks
    sortTasks() {
      const sortField = this.sortBy;
      const ascending = this.sortOrder === "asc";
      
      this.filteredTasks.sort((a, b) => {
        let aVal, bVal;
        
        switch (sortField) {
          case "scheduled_timestamp":
            aVal = parseInt(a.scheduled_timestamp);
            bVal = parseInt(b.scheduled_timestamp);
            break;
          case "task_id":
            aVal = a.task_id;
            bVal = b.task_id;
            break;
          case "status":
            aVal = a.status;
            bVal = b.status;
            break;
          default:
            aVal = a[sortField] || "";
            bVal = b[sortField] || "";
        }
        
        if (aVal < bVal) return ascending ? -1 : 1;
        if (aVal > bVal) return ascending ? 1 : -1;
        return 0;
      });
    },

    // Toggle status filter
    toggleStatusFilter(status) {
      const index = this.selectedStatuses.indexOf(status);
      if (index > -1) {
        this.selectedStatuses.splice(index, 1);
      } else {
        this.selectedStatuses.push(status);
      }
    },

    // Start creating new task
    startCreateTask(template = null) {
      this.isCreatingTask = true;
      
      if (template) {
        this.newTask = {
          scheduledTimestamp: "",
          expiryTimestamp: "",
          gasLimit: 200000,
          gasFee: "0",
          messages: JSON.parse(JSON.stringify(template.messages))
        };
        this.populateAddressesInMessages();
      } else {
        this.newTask = {
          scheduledTimestamp: "",
          expiryTimestamp: "",
          gasLimit: 200000,
          gasFee: "0",
          messages: []
        };
      }
    },

    // Populate addresses in message templates
    populateAddressesInMessages() {
      const walletStore = Alpine.store('walletStore');
      const userAddress = walletStore?.activeWalletMeta?.address;
      const scriptAddress = this.getScriptAddress();
      
      if (!userAddress) return;
      
      this.newTask.messages.forEach(msg => {
        if (msg["@type"] === "/cosmos.bank.v1beta1.MsgSend") {
          msg.from_address = userAddress;
        } else if (msg["@type"] === "/dysonprotocol.script.v1.MsgExec") {
          msg.executor_address = userAddress;
          msg.script_address = scriptAddress;
        }
      });
    },

    // Cancel task creation
    cancelCreateTask() {
      this.isCreatingTask = false;
      this.newTask = {
        scheduledTimestamp: "",
        expiryTimestamp: "",
        gasLimit: 200000,
        gasFee: "0",
        messages: []
      };
    },

    // Add message to new task
    addMessage() {
      this.isAddingMessage = true;
      this.currentMessageType = "bank_send";
      this.newMessage = this.getMessageTemplate(this.currentMessageType);
    },

    // Get message template
    getMessageTemplate(type) {
      const walletStore = Alpine.store('walletStore');
      const userAddress = walletStore?.activeWalletMeta?.address || "";
      const scriptAddress = this.getScriptAddress();
      
      const templates = {
        bank_send: {
          "@type": "/cosmos.bank.v1beta1.MsgSend",
          "from_address": userAddress,
          "to_address": "",
          "amount": [{"denom": "dys", "amount": "1000"}]
        },
        script_exec: {
          "@type": "/dysonprotocol.script.v1.MsgExec",
          "executor_address": userAddress,
          "script_address": scriptAddress,
          "function_name": "",
          "args": "[]",
          "kwargs": "{}",
          "extra_code": "",
          "attached_messages": []
        },
        storage_set: {
          "@type": "/dysonprotocol.storage.v1.MsgStorageSet",
          "owner": scriptAddress,
          "index": "",
          "data": ""
        },
        custom: {}
      };
      
      return JSON.parse(JSON.stringify(templates[type] || templates.custom));
    },

    // Change message type
    changeMessageType(type) {
      this.currentMessageType = type;
      this.newMessage = this.getMessageTemplate(type);
    },

    // Save new message
    saveMessage() {
      if (this.currentMessageType === "custom") {
        try {
          this.newMessage = JSON.parse(this.newMessageJson || "{}");
        } catch (error) {
          alert("Invalid JSON format");
          return;
        }
      }
      
      this.newTask.messages.push(JSON.parse(JSON.stringify(this.newMessage)));
      this.cancelAddMessage();
    },

    // Cancel adding message
    cancelAddMessage() {
      this.isAddingMessage = false;
      this.newMessage = this.getMessageTemplate("bank_send");
      this.newMessageJson = "";
    },

    // Remove message from new task
    removeMessage(index) {
      this.newTask.messages.splice(index, 1);
    },

    // Move message up/down
    moveMessage(index, direction) {
      const newIndex = direction === "up" ? index - 1 : index + 1;
      if (newIndex < 0 || newIndex >= this.newTask.messages.length) return;
      
      const messages = [...this.newTask.messages];
      [messages[index], messages[newIndex]] = [messages[newIndex], messages[index]];
      this.newTask.messages = messages;
    },

    // Parse time input (supports both Unix timestamp and relative time)
    parseTimeInput(input) {
      if (!input.trim()) return null;
      
      // If it's a number, treat as Unix timestamp
      if (/^\d+$/.test(input)) {
        return parseInt(input);
      }
      
      // If it starts with +, treat as relative time
      if (input.startsWith('+')) {
        const now = Math.floor(Date.now() / 1000);
        const relativeSeconds = this.parseRelativeTime(input.substring(1));
        return relativeSeconds ? now + relativeSeconds : null;
      }
      
      // Try to parse as ISO date
      try {
        const date = new Date(input);
        return Math.floor(date.getTime() / 1000);
      } catch {
        return null;
      }
    },

    // Parse relative time (e.g., "1h30m", "2d", "45m")
    parseRelativeTime(input) {
      const regex = /(\d+)([dhms])/g;
      let totalSeconds = 0;
      let match;
      
      while ((match = regex.exec(input)) !== null) {
        const value = parseInt(match[1]);
        const unit = match[2];
        
        switch (unit) {
          case 'd': totalSeconds += value * 24 * 60 * 60; break;
          case 'h': totalSeconds += value * 60 * 60; break;
          case 'm': totalSeconds += value * 60; break;
          case 's': totalSeconds += value; break;
        }
      }
      
      return totalSeconds;
    },

    // Create task
    async createTask() {
      try {
        this.createTaskLoading = true;
        
        const walletStore = Alpine.store('walletStore');
        if (!walletStore?.activeWalletMeta) {
          throw new Error('Wallet not connected');
        }
        
        // Parse timestamps
        const scheduledTimestamp = this.parseTimeInput(this.newTask.scheduledTimestamp);
        if (!scheduledTimestamp) {
          throw new Error('Invalid scheduled timestamp');
        }
        
        let expiryTimestamp = null;
        if (this.newTask.expiryTimestamp.trim()) {
          expiryTimestamp = this.parseTimeInput(this.newTask.expiryTimestamp);
          if (!expiryTimestamp) {
            throw new Error('Invalid expiry timestamp');
          }
        }
        
        // Validate messages
        if (this.newTask.messages.length === 0) {
          throw new Error('At least one message is required');
        }
        
        // Create the task message
        const taskMsg = {
          "@type": "/dysonprotocol.crontask.v1.MsgCreateTask",
          "creator": walletStore.activeWalletMeta.address,
          "scheduled_timestamp": scheduledTimestamp.toString(),
          "expiry_timestamp": expiryTimestamp ? expiryTimestamp.toString() : "0",
          "gas_limit": this.newTask.gasLimit.toString(),
          "gas_fee": this.newTask.gasFee,
          "messages": this.newTask.messages
        };
        
        const result = await walletStore.sendMsg({
          msg: taskMsg,
          gasLimit: Math.max(300000, this.newTask.gasLimit + 100000),
          memo: 'Create scheduled task'
        });
        
        if (result.success) {
          this.cancelCreateTask();
          await this.loadTasks();
        } else {
          console.error("Task creation failed:", result);
          throw new Error(result.rawLog || 'Task creation failed');
        }
        
      } catch (error) {
        console.error("Error creating task:", error);
        alert(`Error creating task: ${error.message}`);
      } finally {
        this.createTaskLoading = false;
      }
    },

    // Delete task
    async deleteTask(taskId) {
      if (!confirm(`Are you sure you want to delete task ${taskId}? This action cannot be undone.`)) {
        return;
      }
      
      try {
        const walletStore = Alpine.store('walletStore');
        if (!walletStore?.activeWalletMeta) {
          throw new Error('Wallet not connected');
        }
        
        const deleteMsg = {
          "@type": "/dysonprotocol.crontask.v1.MsgDeleteTask",
          "creator": walletStore.activeWalletMeta.address,
          "task_id": taskId
        };
        
        const result = await walletStore.sendMsg({
          msg: deleteMsg,
          gasLimit: 200000,
          memo: `Delete task ${taskId}`
        });
        
        if (result.success) {
          await this.loadTasks();
        } else {
          console.error("Task deletion failed:", result);
          throw new Error(result.rawLog || 'Task deletion failed');
        }
        
      } catch (error) {
        console.error("Error deleting task:", error);
        alert(`Error deleting task: ${error.message}`);
      }
    },

    // Clone task
    cloneTask(task) {
      this.startCreateTask();
      this.newTask = {
        scheduledTimestamp: "",
        expiryTimestamp: "",
        gasLimit: parseInt(task.gas_limit),
        gasFee: task.gas_fee,
        messages: JSON.parse(JSON.stringify(task.parsedMessages.map(msg => {
          const { displayType, summary, ...cleanMsg } = msg;
          return cleanMsg;
        })))
      };
    },

    // Show task details
    showDetails(task) {
      this.selectedTask = task;
      this.showTaskDetails = true;
    },

    // Hide task details
    hideDetails() {
      this.selectedTask = null;
      this.showTaskDetails = false;
    },

    // Get status badge class
    getStatusBadgeClass(status) {
      const classes = {
        "PENDING": "badge-info",
        "DONE": "badge-success",
        "FAILED": "badge-error",
        "EXPIRED": "badge-neutral"
      };
      return classes[status] || "badge-neutral";
    },

    // Utility functions
    truncateAddress(address, length = 8) {
      if (!address) return "";
      if (address.length <= length * 2) return address;
      return `${address.slice(0, length)}...${address.slice(-length)}`;
    },

    truncateString(str, length = 30) {
      if (!str) return "";
      if (str.length <= length) return str;
      return `${str.slice(0, length)}...`;
    },

    formatTimestamp(timestamp) {
      if (!timestamp) return "";
      const date = new Date(parseInt(timestamp) * 1000);
      return date.toLocaleString();
    },

    formatGas(gasLimit) {
      return parseInt(gasLimit).toLocaleString();
    },

    // Load more tasks
    loadMore() {
      this.currentPage++;
      this.loadTasks(true);
    },

    // Export tasks
    exportTasks() {
      const exportData = {
        exported_at: new Date().toISOString(),
        creator: this.getScriptAddress(),
        tasks: this.filteredTasks.map(task => ({
          task_id: task.task_id,
          status: task.status,
          scheduled_timestamp: task.scheduled_timestamp,
          expiry_timestamp: task.expiry_timestamp,
          gas_limit: task.gas_limit,
          gas_fee: task.gas_fee,
          messages: task.parsedMessages.map(msg => {
            const { displayType, summary, ...cleanMsg } = msg;
            return cleanMsg;
          })
        }))
      };
      
      const blob = new Blob([JSON.stringify(exportData, null, 2)], { type: 'application/json' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `dyson-tasks-${Date.now()}.json`;
      a.click();
      URL.revokeObjectURL(url);
    }
  }));
}); 