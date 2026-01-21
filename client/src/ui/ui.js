/**
 * 游戏UI管理器
 */
class GameUI {
    constructor(gameEngine, network, wallet) {
        this.gameEngine = gameEngine;
        this.network = network;
        this.wallet = wallet;

        // 好友列表和在线玩家数据
        this.friends = new Map(); // playerId -> { nickname, status: 'online'|'offline'|'idle', level, lastSeen }
        this.lobbyPlayers = new Map(); // playerId -> player data
        this.currentTab = 'friends';

        this.setupEventListeners();
        this.setupNetworkCallbacks();
        this.setupPlayerTabs();

        // 初始化UI状态
        this.updateUI();
    }
    
    setupEventListeners() {
        // DID创建按钮
        const createDIDBtn = document.getElementById('createDIDBtn');
        if (createDIDBtn) {
            createDIDBtn.addEventListener('click', () => this.handleCreateDID());
        }

        // 连接游戏按钮
        const connectBtn = document.getElementById('connectBtn');
        if (connectBtn) {
            connectBtn.addEventListener('click', () => this.handleConnect());
        }

        // 查看钱包按钮
        const viewWalletBtn = document.getElementById('viewWalletBtn');
        if (viewWalletBtn) {
            viewWalletBtn.addEventListener('click', () => this.wallet.showWallet());
        }

        // 聊天输入
        const chatInput = document.getElementById('chatInput');
        if (chatInput) {
            chatInput.addEventListener('keypress', (e) => {
                if (e.key === 'Enter') {
                    this.handleSendMessage();
                }
            });
        }

        // 钱包模态框关闭
        const walletModal = document.getElementById('walletModal');
        if (walletModal) {
            walletModal.addEventListener('click', (e) => {
                if (e.target === walletModal) {
                    walletModal.style.display = 'none';
                }
            });
        }
    }

    setupPlayerTabs() {
        // 玩家列表标签切换
        const tabs = document.querySelectorAll('.player-tab');
        tabs.forEach(tab => {
            tab.addEventListener('click', () => {
                const tabName = tab.dataset.tab;
                this.switchPlayerTab(tabName);
            });
        });
    }

    switchPlayerTab(tabName) {
        this.currentTab = tabName;

        // 更新标签样式
        document.querySelectorAll('.player-tab').forEach(tab => {
            tab.classList.remove('active');
            if (tab.dataset.tab === tabName) {
                tab.classList.add('active');
            }
        });

        // 切换内容区域
        document.querySelectorAll('.tab-content').forEach(content => {
            content.classList.remove('active');
        });
        document.getElementById(`${tabName}Tab`).classList.add('active');

        // 更新玩家列表
        this.updatePlayerList();
    }
    
    setupNetworkCallbacks() {
        // 设置网络回调
        this.network.setGameEngine(this.gameEngine);
        this.network.setWallet(this.wallet);

        this.network.setOnConnect(() => {
            console.log('Connected to game server');
            this.updateConnectionUI(true);
            this.startPlayerListUpdate();
        });

        this.network.setOnDisconnect(() => {
            console.log('Disconnected from game server');
            this.updateConnectionUI(false);
            this.stopPlayerListUpdate();
        });

        this.network.setOnError((error) => {
            console.error('Network error:', error);
            this.showNotification('网络连接错误', 'error');
        });
    }

    startPlayerListUpdate() {
        // 定期更新玩家列表
        this.playerListInterval = setInterval(() => {
            this.updatePlayerList();
        }, 1000);
    }

    stopPlayerListUpdate() {
        if (this.playerListInterval) {
            clearInterval(this.playerListInterval);
            this.playerListInterval = null;
        }
    }

    updatePlayerList() {
        const friendsTab = document.getElementById('friendsTab');
        const lobbyTab = document.getElementById('lobbyTab');
        const playerCount = document.getElementById('playerCount');
        if (!friendsTab || !lobbyTab || !playerCount) return;

        const players = this.gameEngine.gameState.players;

        // 更新大厅玩家 (只显示在线玩家)
        this.lobbyPlayers.clear();
        for (const [playerId, player] of players) {
            if (player.status === 'online') {
                this.lobbyPlayers.set(playerId, player);
            }
        }

        // 更新好友状态 (从在线玩家中查找)
        for (const [friendId, friend] of this.friends) {
            const onlinePlayer = players.get(friendId);
            if (onlinePlayer) {
                friend.status = 'online';
                friend.nickname = onlinePlayer.nickname;
                friend.level = onlinePlayer.level;
            } else {
                friend.status = 'offline';
            }
        }

        // 更新总玩家数
        playerCount.textContent = this.lobbyPlayers.size;

        // 渲染好友标签页
        this.renderFriendsTab(friendsTab);

        // 渲染大厅标签页
        this.renderLobbyTab(lobbyTab);
    }

    renderFriendsTab(tabElement) {
        if (this.friends.size === 0) {
            tabElement.innerHTML = '<div class="no-players">暂无好友</div>';
            return;
        }

        const currentPlayerId = this.gameEngine.gameState.currentPlayer ?
            this.gameEngine.gameState.currentPlayer.id : null;

        let html = '';
        for (const [playerId, friend] of this.friends) {
            const isCurrentPlayer = playerId === currentPlayerId;
            const statusClass = friend.status || 'offline';
            const statusLabel = statusClass === 'online' ? '在线' :
                              statusClass === 'idle' ? '离开' : '离线';

            html += `
                <div class="player-item ${isCurrentPlayer ? 'current-player' : ''}">
                    <div class="player-status ${statusClass}"></div>
                    <div class="player-info">
                        <div class="player-nickname">
                            ${friend.nickname || 'Unknown'}
                            ${isCurrentPlayer ? ' <span class="current-player-badge">(你)</span>' : ''}
                        </div>
                        <div class="player-level">Lv.${friend.level || 1}</div>
                    </div>
                    <div class="player-action">
                        <button onclick="gameUI.sendPrivateMessage('${playerId}')">私信</button>
                    </div>
                </div>
            `;
        }

        tabElement.innerHTML = html;
    }

    renderLobbyTab(tabElement) {
        if (this.lobbyPlayers.size === 0) {
            tabElement.innerHTML = '<div class="no-players">暂无在线玩家</div>';
            return;
        }

        const currentPlayerId = this.gameEngine.gameState.currentPlayer ?
            this.gameEngine.gameState.currentPlayer.id : null;

        let html = '';
        for (const [playerId, player] of this.lobbyPlayers) {
            const isCurrentPlayer = playerId === currentPlayerId;
            const isFriend = this.friends.has(playerId);

            html += `
                <div class="player-item ${isCurrentPlayer ? 'current-player' : ''}">
                    <div class="player-status online"></div>
                    <div class="player-info">
                        <div class="player-nickname">
                            ${player.nickname || 'Unknown'}
                            ${isCurrentPlayer ? ' <span class="current-player-badge">(你)</span>' : ''}
                            ${isFriend ? ' <span class="friend-badge">★好友</span>' : ''}
                        </div>
                        <div class="player-level">Lv.${player.level || 1}</div>
                    </div>
                    <div class="player-action">
                        ${!isFriend && !isCurrentPlayer ? `<button onclick="gameUI.sendFriendRequest('${playerId}')">添加好友</button>` : ''}
                        <button onclick="gameUI.sendPrivateMessage('${playerId}')">私信</button>
                    </div>
                </div>
            `;
        }

        tabElement.innerHTML = html;
    }
    
    async handleCreateDID() {
        try {
            const createDIDBtn = document.getElementById('createDIDBtn');
            if (createDIDBtn) {
                createDIDBtn.disabled = true;
                createDIDBtn.textContent = '创建中...';
            }
            
            // 获取用户输入（这里可以添加一个模态框来收集信息）
            const nickname = prompt('请输入昵称:') || 'Player';
            
            const result = await this.wallet.createDID('default', nickname, 1);
            
            this.showNotification('DID创建成功!', 'success');
            this.updateUI();
            
        } catch (error) {
            console.error('Failed to create DID:', error);
            this.showNotification('DID创建失败: ' + error.message, 'error');
        } finally {
            const createDIDBtn = document.getElementById('createDIDBtn');
            if (createDIDBtn) {
                createDIDBtn.disabled = false;
                createDIDBtn.textContent = '创建身份';
            }
        }
    }
    
    async handleConnect() {
        if (!this.wallet.hasDID()) {
            this.showNotification('请先创建DID身份', 'error');
            return;
        }
        
        try {
            const connectBtn = document.getElementById('connectBtn');
            if (connectBtn) {
                connectBtn.disabled = true;
                connectBtn.textContent = '连接中...';
            }
            
            // 连接到游戏服务器
            await this.network.connect();
            
            // 发送身份认证
            this.network.authenticate(this.wallet.did);
            
            this.showNotification('连接成功!', 'success');
            
        } catch (error) {
            console.error('Failed to connect:', error);
            this.showNotification('连接失败: ' + error.message, 'error');
        } finally {
            const connectBtn = document.getElementById('connectBtn');
            if (connectBtn) {
                connectBtn.disabled = false;
                connectBtn.textContent = '连接游戏';
            }
        }
    }
    
    handleSendMessage() {
        const chatInput = document.getElementById('chatInput');
        if (!chatInput) return;
        
        const message = chatInput.value.trim();
        if (!message) return;
        
        if (!this.network.isConnected()) {
            this.showNotification('请先连接到游戏服务器', 'error');
            return;
        }
        
        // 发送聊天消息
        this.network.sendChatMessage(message);
        
        // 清空输入框
        chatInput.value = '';
    }
    
    updateUI() {
        this.updateDIDUI();
        this.updateWalletUI();
        this.updateConnectionUI(this.network.isConnected());
    }
    
    updateDIDUI() {
        const didElement = document.getElementById('playerDID');
        const createDIDBtn = document.getElementById('createDIDBtn');
        
        if (didElement) {
            if (this.wallet.hasDID()) {
                didElement.textContent = `${this.wallet.did.substring(0, 30)}...`;
                didElement.className = 'status';
            } else {
                didElement.textContent = '未创建';
                didElement.className = 'error';
            }
        }
        
        if (createDIDBtn) {
            createDIDBtn.style.display = this.wallet.hasDID() ? 'none' : 'inline-block';
        }
    }
    
    updateWalletUI() {
        const credentialCount = document.getElementById('credentialCount');
        if (credentialCount) {
            credentialCount.textContent = this.wallet.getCredentialCount();
        }
    }
    
    updateConnectionUI(connected) {
        const statusElement = document.getElementById('connectionStatus');
        const connectBtn = document.getElementById('connectBtn');
        
        if (statusElement) {
            statusElement.textContent = connected ? '已连接' : '未连接';
            statusElement.className = connected ? 'status' : 'error';
        }
        
        if (connectBtn) {
            connectBtn.textContent = connected ? '已连接' : '连接游戏';
            connectBtn.disabled = connected || !this.wallet.hasDID();
        }
    }
    
    showNotification(message, type = 'info') {
        // 创建通知元素
        const notification = document.createElement('div');
        notification.className = `notification ${type}`;
        notification.textContent = message;
        
        // 添加样式
        notification.style.cssText = `
            position: fixed;
            top: 20px;
            right: 20px;
            padding: 10px 20px;
            border-radius: 5px;
            color: white;
            font-weight: bold;
            z-index: 10000;
            animation: slideIn 0.3s ease-out;
        `;
        
        // 根据类型设置背景色
        switch (type) {
            case 'success':
                notification.style.backgroundColor = '#4CAF50';
                break;
            case 'error':
                notification.style.backgroundColor = '#f44336';
                break;
            case 'warning':
                notification.style.backgroundColor = '#ff9800';
                break;
            default:
                notification.style.backgroundColor = '#2196F3';
        }
        
        // 添加到页面
        document.body.appendChild(notification);
        
        // 3秒后自动移除
        setTimeout(() => {
            if (notification.parentNode) {
                notification.style.animation = 'slideOut 0.3s ease-in';
                setTimeout(() => {
                    if (notification.parentNode) {
                        notification.parentNode.removeChild(notification);
                    }
                }, 300);
            }
        }, 3000);
        
        // 点击移除
        notification.addEventListener('click', () => {
            if (notification.parentNode) {
                notification.parentNode.removeChild(notification);
            }
        });
    }
    
    // 游戏状态显示
    updatePlayerPosition(x, y) {
        const playerXElement = document.getElementById('playerX');
        const playerYElement = document.getElementById('playerY');
        
        if (playerXElement) playerXElement.textContent = Math.round(x);
        if (playerYElement) playerYElement.textContent = Math.round(y);
    }
    
    // 任务UI
    showTaskDialog(task) {
        const dialog = this.createTaskDialog(task);
        document.body.appendChild(dialog);
    }
    
    createTaskDialog(task) {
        const dialog = document.createElement('div');
        dialog.className = 'task-dialog';
        dialog.style.cssText = `
            position: fixed;
            top: 50%;
            left: 50%;
            transform: translate(-50%, -50%);
            background: #222;
            border: 2px solid #555;
            border-radius: 10px;
            padding: 20px;
            min-width: 300px;
            z-index: 1000;
            color: white;
        `;
        
        dialog.innerHTML = `
            <h3>${task.name}</h3>
            <p>${task.description}</p>
            <div class=\"task-objectives\">
                <h4>目标:</h4>
                ${task.objectives.map(obj => `
                    <div class=\"objective ${obj.completed ? 'completed' : ''}\">
                        ${obj.description} (${obj.current}/${obj.required})
                    </div>
                `).join('')}
            </div>
            <div class=\"task-rewards\">
                <h4>奖励:</h4>
                ${task.rewards.map(reward => `
                    <div class=\"reward\">
                        ${reward.type}: ${reward.value}
                    </div>
                `).join('')}
            </div>
            <div class=\"task-actions\">
                <button onclick=\"this.parentNode.parentNode.remove()\">关闭</button>
                ${task.status === 'available' ? 
                    `<button onclick="gameUI.acceptTask('${task.id}'); this.parentNode.parentNode.remove()">接受任务</button>` : 
                    ''}
            </div>
        `;
        
        return dialog;
    }
    
    acceptTask(taskId) {
        this.network.sendPlayerAction('accept_task', { taskId: taskId });
        this.showNotification('任务已接受!', 'success');
    }
    
    // 凭证通知
    showCredentialNotification(credential) {
        const notification = document.createElement('div');
        notification.className = 'credential-notification';
        notification.style.cssText = `
            position: fixed;
            top: 50%;
            left: 50%;
            transform: translate(-50%, -50%);
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            border: 2px solid #gold;
            border-radius: 15px;
            padding: 30px;
            text-align: center;
            color: white;
            z-index: 10000;
            animation: credentialPop 0.5s ease-out;
        `;
        
        const types = credential.type ? credential.type.join(', ') : 'Credential';
        
        notification.innerHTML = `
            <div style=\"font-size: 24px; margin-bottom: 10px;\">🏆</div>
            <h3>获得新凭证!</h3>
            <p><strong>${types}</strong></p>
            <button onclick=\"this.parentNode.removeChild(this)\" style=\"
                background: rgba(255,255,255,0.2);
                border: 1px solid rgba(255,255,255,0.3);
                color: white;
                padding: 10px 20px;
                border-radius: 5px;
                cursor: pointer;
                margin-top: 15px;
            \">确定</button>
        `;
        
        document.body.appendChild(notification);
        
        // 5秒后自动移除
        setTimeout(() => {
            if (notification.parentNode) {
                notification.parentNode.removeChild(notification);
            }
        }, 5000);
    }
    
    // 添加CSS动画
    addAnimationStyles() {
        if (document.getElementById('gameUIStyles')) return;

        const style = document.createElement('style');
        style.id = 'gameUIStyles';
        style.textContent = `
            @keyframes slideIn {
                from {
                    transform: translateX(100%);
                    opacity: 0;
                }
                to {
                    transform: translateX(0);
                    opacity: 1;
                }
            }

            @keyframes slideOut {
                from {
                    transform: translateX(0);
                    opacity: 1;
                }
                to {
                    transform: translateX(100%);
                    opacity: 0;
                }
            }

            @keyframes credentialPop {
                0% {
                    transform: translate(-50%, -50%) scale(0.5);
                    opacity: 0;
                }
                50% {
                    transform: translate(-50%, -50%) scale(1.1);
                }
                100% {
                    transform: translate(-50%, -50%) scale(1);
                    opacity: 1;
                }
            }

            .task-dialog .objective.completed {
                color: #4CAF50;
                text-decoration: line-through;
            }

            .credential-item {
                border: 1px solid #555;
                border-radius: 5px;
                padding: 10px;
                margin: 10px 0;
                background: rgba(255,255,255,0.05);
            }

            .credential-header {
                display: flex;
                justify-content: space-between;
                align-items: center;
                margin-bottom: 10px;
            }

            .credential-actions {
                margin-top: 10px;
            }

            .credential-actions button {
                margin-right: 5px;
            }

            .wallet-section {
                margin-bottom: 20px;
                padding-bottom: 15px;
                border-bottom: 1px solid #555;
            }

            .wallet-actions {
                text-align: center;
                margin-top: 20px;
            }

            .wallet-actions button.danger {
                background: #f44336;
            }

            .no-credentials {
                color: #888;
                font-style: italic;
                text-align: center;
                padding: 20px;
            }

            .no-players {
                color: #888;
                font-style: italic;
                text-align: center;
                padding: 20px;
            }

            .player-item {
                transition: background-color 0.2s;
            }

            .player-item:hover {
                background-color: rgba(255,255,255,0.1);
            }

            .player-item.current-player {
                background-color: rgba(33, 150, 243, 0.2);
                border-left: 3px solid #2196F3;
            }

            .player-info {
                flex: 1;
            }

            .player-nickname {
                font-size: 13px;
                color: #fff;
            }

            .player-level {
                font-size: 11px;
                color: #888;
                margin-top: 2px;
            }

            .current-player-badge {
                color: #2196F3;
                font-weight: bold;
            }

            #chatMessages::-webkit-scrollbar {
                width: 8px;
            }

            #chatMessages::-webkit-scrollbar-track {
                background: rgba(0, 0, 0, 0.3);
            }

            #chatMessages::-webkit-scrollbar-thumb {
                background: #555;
                border-radius: 4px;
            }

            #chatMessages::-webkit-scrollbar-thumb:hover {
                background: #777;
            }

            #playerList::-webkit-scrollbar {
                width: 6px;
            }

            #playerList::-webkit-scrollbar-track {
                background: rgba(0, 0, 0, 0.3);
            }

            #playerList::-webkit-scrollbar-thumb {
                background: #555;
                border-radius: 3px;
            }
        `;

        document.head.appendChild(style);
    }
    
    // 初始化
    init() {
        this.addAnimationStyles();
        console.log('Game UI initialized');
    }
}

// 全局变量，供HTML中的onclick使用
let gameUI = null;