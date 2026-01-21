/**
 * 玩家钱包管理器
 */
class PlayerWallet {
    constructor() {
        this.did = null;
        this.privateKey = null;
        this.publicKey = null;
        this.credentials = new Map();
        
        // 从本地存储加载数据
        this.loadFromStorage();
    }
    
    // DID 管理
    async createDID(gameId, nickname = '', level = 1) {
        try {
            const response = await fetch('/api/did/create', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    gameId: gameId,
                    nickname: nickname,
                    level: level
                })
            });
            
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }
            
            const data = await response.json();
            
            // 保存DID信息
            this.did = data.did;
            this.privateKey = data.privateKey;
            this.publicKey = data.publicKey;
            
            // 保存到本地存储
            this.saveToStorage();
            
            // 更新UI
            this.updateDIDDisplay();
            
            console.log('DID created successfully:', this.did);
            return data;
            
        } catch (error) {
            console.error('Failed to create DID:', error);
            throw error;
        }
    }
    
    async resolveDID(did) {
        try {
            const response = await fetch(`/api/did/resolve?did=${encodeURIComponent(did)}`);
            
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }
            
            const data = await response.json();
            return data;
            
        } catch (error) {
            console.error('Failed to resolve DID:', error);
            throw error;
        }
    }
    
    // 凭证管理
    addCredential(credential) {
        if (!credential || !credential.id) {
            console.error('Invalid credential');
            return;
        }
        
        this.credentials.set(credential.id, {
            ...credential,
            receivedAt: new Date().toISOString()
        });
        
        // 保存到本地存储
        this.saveToStorage();
        
        console.log('Credential added:', credential.id);
    }
    
    removeCredential(credentialId) {
        if (this.credentials.has(credentialId)) {
            this.credentials.delete(credentialId);
            this.saveToStorage();
            console.log('Credential removed:', credentialId);
        }
    }
    
    getCredential(credentialId) {
        return this.credentials.get(credentialId);
    }
    
    getAllCredentials() {
        return Array.from(this.credentials.values());
    }
    
    getCredentialsByType(type) {
        return this.getAllCredentials().filter(cred => 
            cred.type && cred.type.includes(type)
        );
    }
    
    getCredentialCount() {
        return this.credentials.size;
    }
    
    // 凭证验证
    async verifyCredential(credential) {
        try {
            const response = await fetch('/api/vc/verify', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    credential: credential
                })
            });
            
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }
            
            const data = await response.json();
            return data;
            
        } catch (error) {
            console.error('Failed to verify credential:', error);
            throw error;
        }
    }
    
    // 数字签名
    async signMessage(message) {
        if (!this.privateKey) {
            throw new Error('No private key available');
        }
        
        // 这里应该使用实际的加密库进行签名
        // 为了演示，我们使用简单的哈希
        const encoder = new TextEncoder();
        const data = encoder.encode(message + this.privateKey);
        const hashBuffer = await crypto.subtle.digest('SHA-256', data);
        const hashArray = Array.from(new Uint8Array(hashBuffer));
        const hashHex = hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
        
        return {
            message: message,
            signature: hashHex,
            publicKey: this.publicKey,
            did: this.did
        };
    }
    
    async verifySignature(signedMessage) {
        // 验证签名的逻辑
        const encoder = new TextEncoder();
        const data = encoder.encode(signedMessage.message + this.privateKey);
        const hashBuffer = await crypto.subtle.digest('SHA-256', data);
        const hashArray = Array.from(new Uint8Array(hashBuffer));
        const expectedHash = hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
        
        return expectedHash === signedMessage.signature;
    }
    
    // 本地存储管理
    saveToStorage() {
        const walletData = {
            did: this.did,
            privateKey: this.privateKey,
            publicKey: this.publicKey,
            credentials: Array.from(this.credentials.entries())
        };
        
        try {
            localStorage.setItem('aries_game_wallet', JSON.stringify(walletData));
        } catch (error) {
            console.error('Failed to save wallet to storage:', error);
        }
    }
    
    loadFromStorage() {
        try {
            const stored = localStorage.getItem('aries_game_wallet');
            if (stored) {
                const walletData = JSON.parse(stored);
                
                this.did = walletData.did;
                this.privateKey = walletData.privateKey;
                this.publicKey = walletData.publicKey;
                
                if (walletData.credentials) {
                    this.credentials = new Map(walletData.credentials);
                }
                
                // 更新UI
                this.updateDIDDisplay();
                
                console.log('Wallet loaded from storage');
            }
        } catch (error) {
            console.error('Failed to load wallet from storage:', error);
        }
    }
    
    clearStorage() {
        localStorage.removeItem('aries_game_wallet');
        this.did = null;
        this.privateKey = null;
        this.publicKey = null;
        this.credentials.clear();
        
        this.updateDIDDisplay();
        console.log('Wallet storage cleared');
    }
    
    // UI 更新
    updateDIDDisplay() {
        const didElement = document.getElementById('playerDID');
        if (didElement) {
            didElement.textContent = this.did ? 
                `${this.did.substring(0, 20)}...` : '未创建';
        }
        
        const countElement = document.getElementById('credentialCount');
        if (countElement) {
            countElement.textContent = this.getCredentialCount();
        }
    }
    
    // 钱包UI显示
    showWallet() {
        const modal = document.getElementById('walletModal');
        const content = document.getElementById('walletContent');
        
        if (!modal || !content) return;
        
        // 构建钱包内容
        let html = `
            <div class=\"wallet-section\">
                <h4>身份信息</h4>
                <p><strong>DID:</strong> ${this.did || '未创建'}</p>
                <p><strong>公钥:</strong> ${this.publicKey ? `${this.publicKey.substring(0, 20)}...` : '无'}</p>
            </div>
            
            <div class=\"wallet-section\">
                <h4>凭证 (${this.getCredentialCount()})</h4>
                <div class=\"credentials-list\">
        `;
        
        if (this.credentials.size === 0) {
            html += '<p class=\"no-credentials\">暂无凭证</p>';
        } else {
            for (const [id, credential] of this.credentials) {
                html += this.renderCredential(credential);
            }
        }
        
        html += `
                </div>
            </div>
            
            <div class=\"wallet-actions\">
                <button onclick=\"wallet.exportWallet()\">导出钱包</button>
                <button onclick=\"wallet.clearStorage()\" class=\"danger\">清空钱包</button>
            </div>
        `;
        
        content.innerHTML = html;
        modal.style.display = 'block';
    }
    
    renderCredential(credential) {
        const types = credential.type ? credential.type.join(', ') : 'Unknown';
        const issuedDate = credential.issuanceDate ? 
            new Date(credential.issuanceDate).toLocaleDateString() : 'Unknown';
        const receivedDate = credential.receivedAt ? 
            new Date(credential.receivedAt).toLocaleDateString() : 'Unknown';
        
        return `
            <div class=\"credential-item\">
                <div class=\"credential-header\">
                    <strong>${types}</strong>
                    <span class=\"credential-date\">${issuedDate}</span>
                </div>
                <div class=\"credential-details\">
                    <p><strong>ID:</strong> ${credential.id}</p>
                    <p><strong>颁发者:</strong> ${credential.issuer?.id || 'Unknown'}</p>
                    <p><strong>接收时间:</strong> ${receivedDate}</p>
                    ${this.renderCredentialSubject(credential.credentialSubject || credential.subject)}
                </div>
                <div class=\"credential-actions\">
                    <button onclick=\"wallet.verifyCredential('${credential.id}')\">验证</button>
                    <button onclick=\"wallet.exportCredential('${credential.id}')\">导出</button>
                </div>
            </div>
        `;
    }
    
    renderCredentialSubject(subject) {
        if (!subject) return '';
        
        // 处理数组格式的subject
        const subjectData = Array.isArray(subject) ? subject[0] : subject;
        
        let html = '<div class=\"credential-subject\">';
        
        if (subjectData.achievement) {
            html += `<p><strong>成就:</strong> ${subjectData.achievement}</p>`;
        }
        if (subjectData.level) {
            html += `<p><strong>等级:</strong> ${subjectData.level}</p>`;
        }
        if (subjectData.score) {
            html += `<p><strong>分数:</strong> ${subjectData.score}</p>`;
        }
        if (subjectData.skills && subjectData.skills.length > 0) {
            html += `<p><strong>技能:</strong> ${subjectData.skills.join(', ')}</p>`;
        }
        
        html += '</div>';
        return html;
    }
    
    // 凭证操作
    async verifyCredential(credentialId) {
        const credential = this.getCredential(credentialId);
        if (!credential) {
            alert('凭证不存在');
            return;
        }
        
        try {
            const result = await this.verifyCredential(credential);
            
            if (result.valid) {
                alert('凭证验证成功！');
            } else {
                alert(`凭证验证失败: ${result.message}`);
            }
        } catch (error) {
            alert(`验证失败: ${error.message}`);
        }
    }
    
    exportCredential(credentialId) {
        const credential = this.getCredential(credentialId);
        if (!credential) {
            alert('凭证不存在');
            return;
        }
        
        const dataStr = JSON.stringify(credential, null, 2);
        const dataBlob = new Blob([dataStr], { type: 'application/json' });
        
        const link = document.createElement('a');
        link.href = URL.createObjectURL(dataBlob);
        link.download = `credential_${credentialId}.json`;
        link.click();
    }
    
    exportWallet() {
        const walletData = {
            did: this.did,
            publicKey: this.publicKey,
            credentials: this.getAllCredentials()
        };
        
        const dataStr = JSON.stringify(walletData, null, 2);
        const dataBlob = new Blob([dataStr], { type: 'application/json' });
        
        const link = document.createElement('a');
        link.href = URL.createObjectURL(dataBlob);
        link.download = 'aries_game_wallet.json';
        link.click();
    }
    
    // 工具方法
    hasDID() {
        return !!this.did;
    }
    
    hasCredentials() {
        return this.credentials.size > 0;
    }
    
    getWalletSummary() {
        return {
            did: this.did,
            credentialCount: this.getCredentialCount(),
            hasPrivateKey: !!this.privateKey,
            achievements: this.getCredentialsByType('AchievementCredential').length,
            levels: this.getCredentialsByType('LevelCredential').length,
            skills: this.getCredentialsByType('SkillCredential').length
        };
    }
}