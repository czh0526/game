/**
 * 面板管理器 - 处理面板拖动、调整大小和持久化存储
 */
class PanelManager {
    constructor() {
        this.panels = new Map();
        this.storageKey = 'gamePanelSettings';
        this.defaultSettings = {
            identity: { x: 10, y: 10, width: 250, height: 'auto' },
            wallet: { x: 10, y: 200, width: 250, height: 'auto' },
            control: { x: 10, y: 390, width: 250, height: 'auto' },
            playerList: { x: 10, y: 10, width: 280, height: 'auto' },
            chat: { x: 10, y: 420, width: 280, height: 'auto' }
        };

        this.init();
    }

    init() {
        // 加载保存的面板设置
        this.loadSettings();
        
        // 初始化所有面板
        this.setupPanels();
        
        // 监听窗口大小变化
        window.addEventListener('resize', () => this.onWindowResize());
        
        // 监听页面卸载,保存设置
        window.addEventListener('beforeunload', () => this.saveSettings());
        
        console.log('PanelManager initialized');
    }

    setupPanels() {
        // 获取所有面板
        const panelElements = document.querySelectorAll('.panel[data-panel]');
        
        panelElements.forEach(panel => {
            const panelId = panel.dataset.panel;
            this.panels.set(panelId, panel);
            
            // 应用保存的设置或默认设置
            this.applySettings(panelId, panel);
            
            // 设置拖动和调整大小
            this.makeDraggable(panel);
            this.makeResizable(panel);
        });
    }

    applySettings(panelId, panel) {
        const settings = this.getSettings(panelId);
        
        // 应用位置
        panel.style.left = settings.x + 'px';
        panel.style.top = settings.y + 'px';
        
        // 应用宽度
        if (settings.width) {
            panel.style.width = settings.width + 'px';
        }
        
        // 应用高度(如果不是auto)
        if (settings.height && settings.height !== 'auto') {
            panel.style.height = settings.height + 'px';
        }
    }

    makeDraggable(panel) {
        const header = panel.querySelector('h3');
        if (!header) return;
        
        let isDragging = false;
        let startX, startY, initialX, initialY;
        
        header.addEventListener('mousedown', (e) => {
            if (e.target.tagName === 'BUTTON') return; // 如果点击的是按钮,不拖动
            
            isDragging = true;
            panel.classList.add('dragging');
            
            startX = e.clientX;
            startY = e.clientY;
            
            const rect = panel.getBoundingClientRect();
            initialX = rect.left;
            initialY = rect.top;
            
            e.preventDefault();
        });
        
        document.addEventListener('mousemove', (e) => {
            if (!isDragging) return;
            
            const dx = e.clientX - startX;
            const dy = e.clientY - startY;
            
            const newX = initialX + dx;
            const newY = initialY + dy;
            
            // 限制在窗口内
            const maxX = window.innerWidth - panel.offsetWidth;
            const maxY = window.innerHeight - panel.offsetHeight;
            
            panel.style.left = Math.max(0, Math.min(newX, maxX)) + 'px';
            panel.style.top = Math.max(0, Math.min(newY, maxY)) + 'px';
        });
        
        document.addEventListener('mouseup', () => {
            if (isDragging) {
                isDragging = false;
                panel.classList.remove('dragging');
                
                // 更新设置
                const panelId = panel.dataset.panel;
                this.updateSetting(panelId, {
                    x: parseInt(panel.style.left),
                    y: parseInt(panel.style.top)
                });
            }
        });
    }

    makeResizable(panel) {
        const handle = panel.querySelector('.resize-handle');
        if (!handle) return;
        
        let isResizing = false;
        let startX, startY, startWidth, startHeight;
        
        handle.addEventListener('mousedown', (e) => {
            isResizing = true;
            panel.classList.add('resizing');
            
            startX = e.clientX;
            startY = e.clientY;
            
            const rect = panel.getBoundingClientRect();
            startWidth = rect.width;
            startHeight = rect.height;
            
            e.preventDefault();
            e.stopPropagation();
        });
        
        document.addEventListener('mousemove', (e) => {
            if (!isResizing) return;
            
            const dx = e.clientX - startX;
            const dy = e.clientY - startY;
            
            const newWidth = Math.max(200, startWidth + dx); // 最小宽度200px
            const newHeight = Math.max(150, startHeight + dy); // 最小高度150px
            
            panel.style.width = newWidth + 'px';
            if (panel.dataset.panel !== 'playerList' && panel.dataset.panel !== 'chat') {
                panel.style.height = newHeight + 'px';
            }
        });
        
        document.addEventListener('mouseup', () => {
            if (isResizing) {
                isResizing = false;
                panel.classList.remove('resizing');
                
                // 更新设置
                const panelId = panel.dataset.panel;
                this.updateSetting(panelId, {
                    width: parseInt(panel.style.width),
                    height: panel.style.height === 'auto' ? 'auto' : parseInt(panel.style.height)
                });
            }
        });
    }

    getSettings(panelId) {
        const allSettings = this.loadSettings();
        return allSettings[panelId] || this.defaultSettings[panelId];
    }

    updateSetting(panelId, updates) {
        const allSettings = this.loadSettings();
        allSettings[panelId] = { ...allSettings[panelId], ...updates };
        this.saveSettings(allSettings);
    }

    loadSettings() {
        try {
            const saved = localStorage.getItem(this.storageKey);
            if (saved) {
                return JSON.parse(saved);
            }
        } catch (error) {
            console.error('Failed to load panel settings:', error);
        }
        return { ...this.defaultSettings };
    }

    saveSettings(settings = null) {
        try {
            const allSettings = settings || this.getCurrentSettings();
            localStorage.setItem(this.storageKey, JSON.stringify(allSettings));
            console.log('Panel settings saved:', allSettings);
        } catch (error) {
            console.error('Failed to save panel settings:', error);
        }
    }

    getCurrentSettings() {
        const settings = {};
        
        this.panels.forEach((panel, panelId) => {
            settings[panelId] = {
                x: parseInt(panel.style.left) || this.defaultSettings[panelId].x,
                y: parseInt(panel.style.top) || this.defaultSettings[panelId].y,
                width: parseInt(panel.style.width) || this.defaultSettings[panelId].width,
                height: panel.style.height === 'auto' ? 'auto' : parseInt(panel.style.height) || this.defaultSettings[panelId].height
            };
        });
        
        return settings;
    }

    onWindowResize() {
        // 确保面板在窗口内
        this.panels.forEach((panel, panelId) => {
            const rect = panel.getBoundingClientRect();
            
            if (rect.right > window.innerWidth) {
                panel.style.left = (window.innerWidth - panel.offsetWidth) + 'px';
                this.updateSetting(panelId, { x: parseInt(panel.style.left) });
            }
            
            if (rect.bottom > window.innerHeight) {
                panel.style.top = (window.innerHeight - panel.offsetHeight) + 'px';
                this.updateSetting(panelId, { y: parseInt(panel.style.top) });
            }
        });
    }

    resetSettings() {
        localStorage.removeItem(this.storageKey);
        location.reload();
    }
}

// 全局实例
let panelManager = null;
