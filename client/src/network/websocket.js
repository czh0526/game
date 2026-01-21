/**
 * WebSocket 网络通信管理器
 */
class GameNetwork {
    constructor() {
        this.ws = null;
        this.connected = false;
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 5;
        this.reconnectDelay = 1000;
        
        // 消息处理器
        this.messageHandlers = new Map();
        
        // 连接状态回调
        this.onConnectCallback = null;
        this.onDisconnectCallback = null;
        this.onErrorCallback = null;
        
        this.setupMessageHandlers();
    }
    
    setupMessageHandlers() {
        // 注册默认消息处理器
        this.registerHandler('auth', (data) => this.handleAuth(data));
        this.registerHandler('join_room', (data) => this.handleJoinRoom(data));
        this.registerHandler('leave_room', (data) => this.handleLeaveRoom(data));
        this.registerHandler('player_move', (data) => this.handlePlayerMove(data));
        this.registerHandler('player_update', (data) => this.handlePlayerUpdate(data));
        this.registerHandler('game_state', (data) => this.handleGameState(data));
        this.registerHandler('task_update', (data) => this.handleTaskUpdate(data));
        this.registerHandler('chat', (data) => this.handleChat(data));
        this.registerHandler('credential', (data) => this.handleCredential(data));
        this.registerHandler('error', (data) => this.handleError(data));
    }
    
    connect(url = null) {
        if (this.connected) {
            console.log('Already connected');
            return Promise.resolve();
        }
        
        return new Promise((resolve, reject) => {
            try {
                // 构建WebSocket URL
                const wsUrl = url || this.buildWebSocketURL();
                console.log('Connecting to:', wsUrl);
                
                this.ws = new WebSocket(wsUrl);
                
                this.ws.onopen = () => {
                    console.log('WebSocket connected');
                    this.connected = true;
                    this.reconnectAttempts = 0;
                    this.updateConnectionStatus('已连接');
                    
                    if (this.onConnectCallback) {
                        this.onConnectCallback();
                    }
                    
                    resolve();
                };
                
                this.ws.onmessage = (event) => {
                    try {
                        const message = JSON.parse(event.data);
                        this.handleMessage(message);
                    } catch (error) {
                        console.error('Failed to parse message:', error);
                    }
                };
                
                this.ws.onclose = (event) => {
                    console.log('WebSocket disconnected:', event.code, event.reason);
                    this.connected = false;
                    this.updateConnectionStatus('已断开');
                    
                    if (this.onDisconnectCallback) {
                        this.onDisconnectCallback(event);
                    }
                    
                    // 尝试重连
                    if (this.reconnectAttempts < this.maxReconnectAttempts) {
                        this.scheduleReconnect();
                    }
                };
                
                this.ws.onerror = (error) => {
                    console.error('WebSocket error:', error);
                    this.updateConnectionStatus('连接错误');
                    
                    if (this.onErrorCallback) {
                        this.onErrorCallback(error);
                    }
                    
                    reject(error);
                };
                
            } catch (error) {
                console.error('Failed to create WebSocket:', error);
                reject(error);
            }
        });
    }
    
    buildWebSocketURL() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const host = window.location.host;
        return `${protocol}//${host}/ws/game`;
    }
    
    disconnect() {
        if (this.ws) {
            this.ws.close();
            this.ws = null;
        }
        this.connected = false;
        this.updateConnectionStatus('未连接');
    }
    
    scheduleReconnect() {
        this.reconnectAttempts++;
        const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);
        
        console.log(`Scheduling reconnect attempt ${this.reconnectAttempts} in ${delay}ms`);
        this.updateConnectionStatus(`重连中... (${this.reconnectAttempts}/${this.maxReconnectAttempts})`);
        
        setTimeout(() => {
            if (!this.connected) {
                this.connect();
            }
        }, delay);
    }
    
    send(type, data = {}, roomId = null, playerId = null) {
        if (!this.connected || !this.ws) {
            console.error('Cannot send message: not connected');
            return false;
        }
        
        const message = {
            type: type,
            data: data,
            timestamp: new Date().toISOString()
        };
        
        if (roomId) message.roomId = roomId;
        if (playerId) message.playerId = playerId;
        
        try {
            this.ws.send(JSON.stringify(message));
            return true;
        } catch (error) {
            console.error('Failed to send message:', error);
            return false;
        }
    }
    
    handleMessage(message) {
        console.log('Received message:', message);
        
        const handler = this.messageHandlers.get(message.type);
        if (handler) {
            handler(message);
        } else {
            console.warn('No handler for message type:', message.type);
        }
    }
    
    registerHandler(type, handler) {
        this.messageHandlers.set(type, handler);
    }
    
    // 消息处理器
    handleAuth(message) {
        console.log('Auth response:', message.data);
        
        if (message.data.success) {
            // 认证成功，自动加入默认房间
            this.joinRoom('default');
        } else {
            console.error('Authentication failed');
        }
    }
    
    handleJoinRoom(message) {
        console.log('Joined room:', message.data);
        
        if (message.data.success && this.gameEngine) {
            // 设置游戏状态
            const room = message.data.room;
            const gameState = message.data.gameState;
            
            if (gameState.map) {
                this.gameEngine.setGameMap(gameState.map);
            }
            
            if (gameState.tasks) {
                this.gameEngine.setTasks(gameState.tasks);
            }
            
            // 添加地图对象
            if (gameState.map && gameState.map.objects) {
                gameState.map.objects.forEach(obj => {
                    this.gameEngine.addGameObject(obj);
                });
            }
        }
    }
    
    handleLeaveRoom(message) {
        console.log('Left room:', message.data);
    }
    
    handlePlayerMove(message) {
        if (this.gameEngine && message.playerId) {
            this.gameEngine.updatePlayer(message.playerId, {
                position: message.data.position
            });
        }
    }
    
    handlePlayerUpdate(message) {
        if (!this.gameEngine) return;
        
        const action = message.data.action;
        const player = message.data.player;
        
        switch (action) {
            case 'joined':
                this.gameEngine.addPlayer(player);
                this.addChatMessage(`${player.nickname} 加入了游戏`, 'info');
                break;
            case 'left':
                this.gameEngine.removePlayer(player.id);
                this.addChatMessage(`${player.nickname} 离开了游戏`, 'info');
                break;
            case 'disconnected':
                this.gameEngine.updatePlayer(player.id, { status: 'offline' });
                this.addChatMessage(`${player.nickname} 断开连接`, 'info');
                break;
        }
    }
    
    handleGameState(message) {
        console.log('Game state update:', message.data);
        // 处理游戏状态更新
    }
    
    handleTaskUpdate(message) {
        if (!this.gameEngine) return;
        
        const task = message.data.task;
        const action = message.data.action;
        
        this.gameEngine.updateTask(task.id, task);
        
        if (action === 'completed' && message.playerId) {
            const player = this.gameEngine.gameState.players.get(message.playerId);
            if (player) {
                this.addChatMessage(`${player.nickname} 完成了任务: ${task.name}`, 'success');
            }
        }
    }
    
    handleChat(message) {
        const nickname = message.data.nickname || 'Unknown';
        const text = message.data.message;
        
        this.addChatMessage(`${nickname}: ${text}`, 'chat');
    }
    
    handleCredential(message) {
        console.log('Received credential:', message.data);
        
        // 通知钱包管理器
        if (this.wallet) {
            this.wallet.addCredential(message.data.credential);
        }
        
        // 显示通知
        this.addChatMessage(message.data.message || '获得新凭证!', 'credential');
        
        // 更新凭证计数
        this.updateCredentialCount();
    }
    
    handleError(message) {
        console.error('Server error:', message.data.message);
        this.addChatMessage(`错误: ${message.data.message}`, 'error');
    }
    
    // 发送消息的便捷方法
    authenticate(did) {
        return this.send('auth', { did: did });
    }
    
    joinRoom(roomId) {
        return this.send('join_room', { roomId: roomId });
    }
    
    leaveRoom() {
        return this.send('leave_room');
    }
    
    sendPlayerMove(position) {
        return this.send('player_move', {
            x: position.x,
            y: position.y
        });
    }
    
    sendPlayerAction(action, data = {}) {
        return this.send('player_action', {
            action: action,
            ...data
        });
    }
    
    sendChatMessage(message) {
        return this.send('chat', { message: message });
    }
    
    // UI 更新方法
    updateConnectionStatus(status) {
        const statusElement = document.getElementById('connectionStatus');
        if (statusElement) {
            statusElement.textContent = status;
            statusElement.className = this.connected ? 'status' : 'error';
        }
    }
    
    addChatMessage(message, type = 'info') {
        const chatMessages = document.getElementById('chatMessages');
        if (!chatMessages) return;
        
        const messageElement = document.createElement('div');
        messageElement.className = type;
        messageElement.textContent = `[${new Date().toLocaleTimeString()}] ${message}`;
        
        chatMessages.appendChild(messageElement);
        chatMessages.scrollTop = chatMessages.scrollHeight;
        
        // 限制消息数量
        while (chatMessages.children.length > 100) {
            chatMessages.removeChild(chatMessages.firstChild);
        }
    }
    
    updateCredentialCount() {
        const countElement = document.getElementById('credentialCount');
        if (countElement && this.wallet) {
            countElement.textContent = this.wallet.getCredentialCount();
        }
    }
    
    // 设置回调
    setGameEngine(gameEngine) {
        this.gameEngine = gameEngine;
        
        // 设置游戏引擎回调
        gameEngine.setPlayerMoveCallback((position) => {
            this.sendPlayerMove(position);
        });
        
        gameEngine.setObjectInteractCallback((obj) => {
            this.sendPlayerAction('interact', { objectId: obj.id });
        });
    }
    
    setWallet(wallet) {
        this.wallet = wallet;
    }
    
    setOnConnect(callback) {
        this.onConnectCallback = callback;
    }
    
    setOnDisconnect(callback) {
        this.onDisconnectCallback = callback;
    }
    
    setOnError(callback) {
        this.onErrorCallback = callback;
    }
    
    // 状态查询
    isConnected() {
        return this.connected;
    }
    
    getConnectionState() {
        return {
            connected: this.connected,
            reconnectAttempts: this.reconnectAttempts,
            maxReconnectAttempts: this.maxReconnectAttempts
        };
    }
}