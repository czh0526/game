/**
 * HTML5 Canvas 游戏引擎
 */
class GameEngine {
    constructor(canvasId) {
        this.canvas = document.getElementById(canvasId);
        this.ctx = this.canvas.getContext('2d');
        this.running = false;
        this.lastTime = 0;
        
        // 游戏状态
        this.gameState = {
            players: new Map(),
            currentPlayer: null,
            map: null,
            tasks: [],
            objects: []
        };
        
        // 渲染设置
        this.camera = {
            x: 0,
            y: 0,
            zoom: 1
        };
        
        // 输入处理
        this.keys = {};
        this.mouse = { x: 0, y: 0, clicked: false };
        
        this.initCanvas();
        this.setupEventListeners();
    }
    
    initCanvas() {
        // 设置画布大小
        this.resizeCanvas();
        window.addEventListener('resize', () => this.resizeCanvas());
        
        // 设置画布样式
        this.canvas.style.cursor = 'crosshair';
    }
    
    resizeCanvas() {
        const container = this.canvas.parentElement;
        this.canvas.width = container.clientWidth;
        this.canvas.height = container.clientHeight;
        
        // 更新相机以保持居中
        this.camera.x = this.canvas.width / 2;
        this.camera.y = this.canvas.height / 2;
    }
    
    setupEventListeners() {
        // 键盘事件
        document.addEventListener('keydown', (e) => {
            this.keys[e.code] = true;
            this.handleKeyDown(e);
        });
        
        document.addEventListener('keyup', (e) => {
            this.keys[e.code] = false;
        });
        
        // 鼠标事件
        this.canvas.addEventListener('mousemove', (e) => {
            const rect = this.canvas.getBoundingClientRect();
            this.mouse.x = e.clientX - rect.left;
            this.mouse.y = e.clientY - rect.top;
        });
        
        this.canvas.addEventListener('click', (e) => {
            this.mouse.clicked = true;
            this.handleMouseClick(e);
        });
        
        // 防止右键菜单
        this.canvas.addEventListener('contextmenu', (e) => {
            e.preventDefault();
        });
    }
    
    handleKeyDown(e) {
        // 处理特殊按键
        switch(e.code) {
            case 'Enter':
                // 聚焦到聊天输入框
                const chatInput = document.getElementById('chatInput');
                if (chatInput) {
                    chatInput.focus();
                }
                break;
            case 'Escape':
                // 取消当前操作
                this.cancelCurrentAction();
                break;
        }
    }
    
    handleMouseClick(e) {
        const worldPos = this.screenToWorld(this.mouse.x, this.mouse.y);
        
        // 检查是否点击了游戏对象
        const clickedObject = this.getObjectAtPosition(worldPos.x, worldPos.y);
        if (clickedObject) {
            this.interactWithObject(clickedObject);
        } else {
            // 移动到点击位置
            this.movePlayerTo(worldPos.x, worldPos.y);
        }
        
        this.mouse.clicked = false;
    }
    
    start() {
        if (this.running) return;
        
        this.running = true;
        this.lastTime = performance.now();
        this.gameLoop();
        
        console.log('Game engine started');
    }
    
    stop() {
        this.running = false;
        console.log('Game engine stopped');
    }
    
    gameLoop(currentTime = performance.now()) {
        if (!this.running) return;
        
        const deltaTime = currentTime - this.lastTime;
        this.lastTime = currentTime;
        
        // 更新游戏逻辑
        this.update(deltaTime);
        
        // 渲染游戏
        this.render();
        
        // 继续循环
        requestAnimationFrame((time) => this.gameLoop(time));
    }
    
    update(deltaTime) {
        // 处理玩家输入
        this.handleInput(deltaTime);
        
        // 更新玩家位置
        this.updatePlayers(deltaTime);
        
        // 更新游戏对象
        this.updateObjects(deltaTime);
        
        // 更新相机
        this.updateCamera(deltaTime);
    }
    
    handleInput(deltaTime) {
        if (!this.gameState.currentPlayer) return;
        
        const player = this.gameState.currentPlayer;
        const speed = 200; // 像素/秒
        const moveDistance = speed * (deltaTime / 1000);
        
        let moved = false;
        
        // WASD 移动
        if (this.keys['KeyW'] || this.keys['ArrowUp']) {
            player.position.y -= moveDistance;
            moved = true;
        }
        if (this.keys['KeyS'] || this.keys['ArrowDown']) {
            player.position.y += moveDistance;
            moved = true;
        }
        if (this.keys['KeyA'] || this.keys['ArrowLeft']) {
            player.position.x -= moveDistance;
            moved = true;
        }
        if (this.keys['KeyD'] || this.keys['ArrowRight']) {
            player.position.x += moveDistance;
            moved = true;
        }
        
        // 如果玩家移动了，发送位置更新
        if (moved) {
            this.onPlayerMove(player.position);
        }
    }
    
    updatePlayers(deltaTime) {
        // 更新所有玩家的动画和状态
        for (const [playerId, player] of this.gameState.players) {
            // 这里可以添加玩家动画逻辑
            this.updatePlayerAnimation(player, deltaTime);
        }
    }
    
    updatePlayerAnimation(player, deltaTime) {
        // 简单的呼吸动画效果
        if (!player.animationTime) player.animationTime = 0;
        player.animationTime += deltaTime;
        
        player.animationOffset = Math.sin(player.animationTime * 0.003) * 2;
    }
    
    updateObjects(deltaTime) {
        // 更新游戏对象
        for (const obj of this.gameState.objects) {
            if (obj.update) {
                obj.update(deltaTime);
            }
        }
    }
    
    updateCamera(deltaTime) {
        if (!this.gameState.currentPlayer) return;
        
        const player = this.gameState.currentPlayer;
        const targetX = this.canvas.width / 2 - player.position.x;
        const targetY = this.canvas.height / 2 - player.position.y;
        
        // 平滑相机跟随
        const lerpFactor = 0.1;
        this.camera.x += (targetX - this.camera.x) * lerpFactor;
        this.camera.y += (targetY - this.camera.y) * lerpFactor;
    }
    
    render() {
        // 清空画布
        this.ctx.clearRect(0, 0, this.canvas.width, this.canvas.height);
        
        // 保存上下文状态
        this.ctx.save();
        
        // 应用相机变换
        this.ctx.translate(this.camera.x, this.camera.y);
        this.ctx.scale(this.camera.zoom, this.camera.zoom);
        
        // 渲染游戏世界
        this.renderBackground();
        this.renderMap();
        this.renderObjects();
        this.renderPlayers();
        this.renderUI();
        
        // 恢复上下文状态
        this.ctx.restore();
        
        // 渲染HUD（不受相机影响）
        this.renderHUD();
    }
    
    renderBackground() {
        // 渲染背景
        this.ctx.fillStyle = '#1a1a2e';
        this.ctx.fillRect(-1000, -1000, 2000, 2000);
        
        // 渲染网格
        this.renderGrid();
    }
    
    renderGrid() {
        const gridSize = 50;
        const startX = Math.floor(-this.camera.x / gridSize) * gridSize;
        const startY = Math.floor(-this.camera.y / gridSize) * gridSize;
        const endX = startX + this.canvas.width + gridSize;
        const endY = startY + this.canvas.height + gridSize;
        
        this.ctx.strokeStyle = '#16213e';
        this.ctx.lineWidth = 1;
        
        // 垂直线
        for (let x = startX; x < endX; x += gridSize) {
            this.ctx.beginPath();
            this.ctx.moveTo(x, startY);
            this.ctx.lineTo(x, endY);
            this.ctx.stroke();
        }
        
        // 水平线
        for (let y = startY; y < endY; y += gridSize) {
            this.ctx.beginPath();
            this.ctx.moveTo(startX, y);
            this.ctx.lineTo(endX, y);
            this.ctx.stroke();
        }
    }
    
    renderMap() {
        if (!this.gameState.map) return;
        
        // 渲染地图瓦片
        const map = this.gameState.map;
        const tileSize = 32;
        
        for (let y = 0; y < map.height / tileSize; y++) {
            for (let x = 0; x < map.width / tileSize; x++) {
                // 这里可以根据瓦片类型渲染不同的图案
                this.renderTile(x * tileSize, y * tileSize, tileSize);
            }
        }
    }
    
    renderTile(x, y, size) {
        // 简单的瓦片渲染
        this.ctx.fillStyle = '#0f3460';
        this.ctx.fillRect(x, y, size, size);
        
        this.ctx.strokeStyle = '#16213e';
        this.ctx.lineWidth = 1;
        this.ctx.strokeRect(x, y, size, size);
    }
    
    renderObjects() {
        // 渲染游戏对象
        for (const obj of this.gameState.objects) {
            this.renderObject(obj);
        }
    }
    
    renderObject(obj) {
        this.ctx.save();
        
        // 移动到对象位置
        this.ctx.translate(obj.position.x, obj.position.y);
        
        // 根据对象类型渲染
        switch (obj.type) {
            case 'spawn_point':
                this.renderSpawnPoint(obj);
                break;
            case 'task_giver':
                this.renderTaskGiver(obj);
                break;
            case 'collectible':
                this.renderCollectible(obj);
                break;
            default:
                this.renderDefaultObject(obj);
        }
        
        this.ctx.restore();
    }
    
    renderSpawnPoint(obj) {
        // 渲染出生点
        this.ctx.fillStyle = '#00ff00';
        this.ctx.beginPath();
        this.ctx.arc(0, 0, 15, 0, Math.PI * 2);
        this.ctx.fill();
        
        this.ctx.strokeStyle = '#ffffff';
        this.ctx.lineWidth = 2;
        this.ctx.stroke();
    }
    
    renderTaskGiver(obj) {
        // 渲染任务发布者
        this.ctx.fillStyle = '#ffff00';
        this.ctx.fillRect(-10, -15, 20, 30);
        
        // 感叹号
        this.ctx.fillStyle = '#000000';
        this.ctx.font = '16px Arial';
        this.ctx.textAlign = 'center';
        this.ctx.fillText('!', 0, 5);
    }
    
    renderCollectible(obj) {
        // 渲染可收集物品
        const time = performance.now() * 0.005;
        const bounce = Math.sin(time) * 3;
        
        this.ctx.fillStyle = '#ff6b6b';
        this.ctx.beginPath();
        this.ctx.arc(0, bounce, 8, 0, Math.PI * 2);
        this.ctx.fill();
        
        // 光晕效果
        this.ctx.fillStyle = 'rgba(255, 107, 107, 0.3)';
        this.ctx.beginPath();
        this.ctx.arc(0, bounce, 12, 0, Math.PI * 2);
        this.ctx.fill();
    }
    
    renderDefaultObject(obj) {
        // 默认对象渲染
        this.ctx.fillStyle = '#888888';
        this.ctx.fillRect(-obj.width/2, -obj.height/2, obj.width, obj.height);
    }
    
    renderPlayers() {
        // 渲染所有玩家
        for (const [playerId, player] of this.gameState.players) {
            this.renderPlayer(player);
        }
    }
    
    renderPlayer(player) {
        this.ctx.save();
        
        // 移动到玩家位置
        this.ctx.translate(player.position.x, player.position.y + (player.animationOffset || 0));
        
        // 玩家身体
        const isCurrentPlayer = player === this.gameState.currentPlayer;
        this.ctx.fillStyle = isCurrentPlayer ? '#4ecdc4' : '#45b7aa';
        this.ctx.beginPath();
        this.ctx.arc(0, 0, 15, 0, Math.PI * 2);
        this.ctx.fill();
        
        // 玩家边框
        this.ctx.strokeStyle = isCurrentPlayer ? '#ffffff' : '#cccccc';
        this.ctx.lineWidth = 2;
        this.ctx.stroke();
        
        // 玩家昵称
        this.ctx.fillStyle = '#ffffff';
        this.ctx.font = '12px Arial';
        this.ctx.textAlign = 'center';
        this.ctx.fillText(player.nickname || 'Player', 0, -25);
        
        // 等级显示
        if (player.level) {
            this.ctx.fillStyle = '#ffff00';
            this.ctx.font = '10px Arial';
            this.ctx.fillText(`Lv.${player.level}`, 0, -35);
        }
        
        // 健康条
        if (player.health !== undefined && player.maxHealth) {
            this.renderHealthBar(player);
        }
        
        this.ctx.restore();
    }
    
    renderHealthBar(player) {
        const barWidth = 30;
        const barHeight = 4;
        const healthPercent = player.health / player.maxHealth;
        
        // 背景
        this.ctx.fillStyle = '#333333';
        this.ctx.fillRect(-barWidth/2, 20, barWidth, barHeight);
        
        // 健康值
        this.ctx.fillStyle = healthPercent > 0.5 ? '#00ff00' : 
                            healthPercent > 0.25 ? '#ffff00' : '#ff0000';
        this.ctx.fillRect(-barWidth/2, 20, barWidth * healthPercent, barHeight);
        
        // 边框
        this.ctx.strokeStyle = '#ffffff';
        this.ctx.lineWidth = 1;
        this.ctx.strokeRect(-barWidth/2, 20, barWidth, barHeight);
    }
    
    renderUI() {
        // 渲染游戏内UI元素（受相机影响）
        this.renderTasks();
    }
    
    renderTasks() {
        // 渲染任务指示器
        for (const task of this.gameState.tasks) {
            if (task.status === 'active' && task.position) {
                this.renderTaskIndicator(task);
            }
        }
    }
    
    renderTaskIndicator(task) {
        this.ctx.save();
        
        this.ctx.translate(task.position.x, task.position.y);
        
        // 任务标记
        this.ctx.fillStyle = '#ff9f43';
        this.ctx.beginPath();
        this.ctx.arc(0, -40, 8, 0, Math.PI * 2);
        this.ctx.fill();
        
        // 任务名称
        this.ctx.fillStyle = '#ffffff';
        this.ctx.font = '10px Arial';
        this.ctx.textAlign = 'center';
        this.ctx.fillText(task.name, 0, -50);
        
        this.ctx.restore();
    }
    
    renderHUD() {
        // 渲染HUD元素（不受相机影响）
        this.renderMiniMap();
        this.renderDebugInfo();
    }
    
    renderMiniMap() {
        if (!this.gameState.currentPlayer) return;
        
        const miniMapSize = 100;
        const miniMapX = this.canvas.width - miniMapSize - 10;
        const miniMapY = 10;
        
        this.ctx.save();
        
        // 小地图背景
        this.ctx.fillStyle = 'rgba(0, 0, 0, 0.7)';
        this.ctx.fillRect(miniMapX, miniMapY, miniMapSize, miniMapSize);
        
        this.ctx.strokeStyle = '#ffffff';
        this.ctx.lineWidth = 1;
        this.ctx.strokeRect(miniMapX, miniMapY, miniMapSize, miniMapSize);
        
        // 玩家位置
        const player = this.gameState.currentPlayer;
        const playerX = miniMapX + miniMapSize / 2;
        const playerY = miniMapY + miniMapSize / 2;
        
        this.ctx.fillStyle = '#4ecdc4';
        this.ctx.beginPath();
        this.ctx.arc(playerX, playerY, 3, 0, Math.PI * 2);
        this.ctx.fill();
        
        this.ctx.restore();
    }
    
    renderDebugInfo() {
        if (!this.gameState.currentPlayer) return;
        
        const player = this.gameState.currentPlayer;
        const debugInfo = [
            `Position: (${Math.round(player.position.x)}, ${Math.round(player.position.y)})`,
            `Players: ${this.gameState.players.size}`,
            `FPS: ${Math.round(1000 / (performance.now() - this.lastTime))}`
        ];
        
        this.ctx.save();
        this.ctx.fillStyle = 'rgba(0, 0, 0, 0.7)';
        this.ctx.fillRect(10, this.canvas.height - 80, 200, 70);
        
        this.ctx.fillStyle = '#ffffff';
        this.ctx.font = '12px monospace';
        this.ctx.textAlign = 'left';
        
        debugInfo.forEach((info, index) => {
            this.ctx.fillText(info, 15, this.canvas.height - 60 + index * 15);
        });
        
        this.ctx.restore();
    }
    
    // 工具方法
    screenToWorld(screenX, screenY) {
        return {
            x: (screenX - this.camera.x) / this.camera.zoom,
            y: (screenY - this.camera.y) / this.camera.zoom
        };
    }
    
    worldToScreen(worldX, worldY) {
        return {
            x: worldX * this.camera.zoom + this.camera.x,
            y: worldY * this.camera.zoom + this.camera.y
        };
    }
    
    getObjectAtPosition(x, y) {
        for (const obj of this.gameState.objects) {
            const dx = x - obj.position.x;
            const dy = y - obj.position.y;
            const distance = Math.sqrt(dx * dx + dy * dy);
            
            if (distance < (obj.radius || 20)) {
                return obj;
            }
        }
        return null;
    }
    
    // 游戏事件处理
    onPlayerMove(position) {
        // 更新UI显示
        const playerXElement = document.getElementById('playerX');
        const playerYElement = document.getElementById('playerY');
        
        if (playerXElement) playerXElement.textContent = Math.round(position.x);
        if (playerYElement) playerYElement.textContent = Math.round(position.y);
        
        // 通知网络层
        if (this.onPlayerMoveCallback) {
            this.onPlayerMoveCallback(position);
        }
    }
    
    movePlayerTo(x, y) {
        if (!this.gameState.currentPlayer) return;
        
        this.gameState.currentPlayer.position.x = x;
        this.gameState.currentPlayer.position.y = y;
        
        this.onPlayerMove(this.gameState.currentPlayer.position);
    }
    
    interactWithObject(obj) {
        console.log('Interacting with object:', obj);
        
        if (this.onObjectInteractCallback) {
            this.onObjectInteractCallback(obj);
        }
    }
    
    cancelCurrentAction() {
        // 取消当前操作
        console.log('Action cancelled');
    }
    
    // 游戏状态管理
    setCurrentPlayer(player) {
        this.gameState.currentPlayer = player;
        console.log('Current player set:', player);
    }
    
    addPlayer(player) {
        this.gameState.players.set(player.id, player);
        console.log('Player added:', player);
    }
    
    removePlayer(playerId) {
        this.gameState.players.delete(playerId);
        console.log('Player removed:', playerId);
    }
    
    updatePlayer(playerId, updates) {
        const player = this.gameState.players.get(playerId);
        if (player) {
            Object.assign(player, updates);
        }
    }
    
    setGameMap(map) {
        this.gameState.map = map;
        console.log('Game map set:', map);
    }
    
    addGameObject(obj) {
        this.gameState.objects.push(obj);
    }
    
    removeGameObject(objId) {
        this.gameState.objects = this.gameState.objects.filter(obj => obj.id !== objId);
    }
    
    setTasks(tasks) {
        this.gameState.tasks = tasks;
    }
    
    updateTask(taskId, updates) {
        const task = this.gameState.tasks.find(t => t.id === taskId);
        if (task) {
            Object.assign(task, updates);
        }
    }
    
    // 回调设置
    setPlayerMoveCallback(callback) {
        this.onPlayerMoveCallback = callback;
    }
    
    setObjectInteractCallback(callback) {
        this.onObjectInteractCallback = callback;
    }
}