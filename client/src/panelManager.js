/**
 * 面板管理器 - 处理面板拖动、调整大小和持久化存储
 */
class PanelManager {
    constructor() {
        this.panels = new Map();
        this.storageKey = 'gamePanelSettings';
        this.defaultSettings = {
            identity: { width: 250, height: 'auto', collapsed: false },
            wallet: { width: 250, height: 'auto', collapsed: false },
            control: { width: 250, height: 'auto', collapsed: false },
            playerList: { width: 280, height: 400, collapsed: false },
            chat: { width: 280, height: 300, collapsed: false }
        };

        this.init();
    }

    init() {
        // 清理旧的 localStorage 数据
        this.clearOldSettings();

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

    clearOldSettings() {
        // 暂时禁用清理,调试布局问题
        try {
            const saved = localStorage.getItem(this.storageKey);
            if (saved) {
                localStorage.removeItem(this.storageKey);
                console.log('Cleared all panel settings for fresh start');
            }
        } catch (error) {
            console.error('Failed to clear settings:', error);
        }
    }

    setupPanels() {
        // 获取所有面板
        const panelElements = document.querySelectorAll('.panel[data-panel]');

        panelElements.forEach(panel => {
            const panelId = panel.dataset.panel;
            this.panels.set(panelId, panel);

            // 应用保存的设置或默认设置
            this.applySettings(panelId, panel);

            // 设置调整大小(流式布局不需要拖动)
            this.makeResizable(panel);

            // 设置折叠功能
            this.makeCollapsible(panel);
        });
    }

    applySettings(panelId, panel) {
        const settings = this.getSettings(panelId);

        // 应用宽度
        if (settings.width) {
            panel.style.width = settings.width + 'px';
        }

        // 应用高度
        if (settings.height && settings.height !== 'auto') {
            panel.style.height = settings.height + 'px';
        }

        // 应用折叠状态
        if (settings.collapsed) {
            panel.classList.add('collapsed');
        } else {
            panel.classList.remove('collapsed');
        }
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
            panel.style.height = newHeight + 'px';
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

    makeCollapsible(panel) {
        const collapseButton = panel.querySelector('.collapse-button');
        if (!collapseButton) return;

        collapseButton.addEventListener('click', (e) => {
            e.preventDefault();
            e.stopPropagation();

            const panelId = panel.dataset.panel;
            const settings = this.getSettings(panelId);
            const isCollapsed = !settings.collapsed;

            // 切换折叠状态
            if (isCollapsed) {
                panel.classList.add('collapsed');
            } else {
                panel.classList.remove('collapsed');
            }

            // 更新设置
            this.updateSetting(panelId, { collapsed: isCollapsed });

            console.log(`Panel ${panelId} ${isCollapsed ? 'collapsed' : 'expanded'}`);
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
            const defaultSettings = this.defaultSettings[panelId];

            settings[panelId] = {
                width: parseInt(panel.style.width) || defaultSettings.width,
                height: panel.style.height === 'auto' ? 'auto' : parseInt(panel.style.height) || defaultSettings.height,
                collapsed: panel.classList.contains('collapsed')
            };
        });

        return settings;
    }

    onWindowResize() {
        // 使用相对定位后,面板会自动适应窗口大小,不需要手动调整
        // 此方法保留为空,以备将来需要特殊处理时使用
        console.log('Window resized - panels will automatically adjust due to relative positioning');
    }

    resetSettings() {
        localStorage.removeItem(this.storageKey);
        location.reload();
    }
}

// 全局实例
let panelManager = null;
