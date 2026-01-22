/**
 * 加密工具库 - 使用 Web Crypto API 实现 DID 相关的加密功能
 */
class CryptoUtils {
    constructor() {
        this.algorithm = 'Ed25519';
    }

    /**
     * 生成 Ed25519 密钥对
     * @returns {Promise<{publicKey: string, privateKey: string, keyPair: CryptoKeyPair}>}
     */
    async generateKeyPair() {
        try {
            // Web Crypto API 对 Ed25519 的支持有限，我们使用 X25519 进行密钥交换
            // 对于 Ed25519 签名，我们使用浏览器兼容的方式

            // 生成随机种子
            const seed = new Uint8Array(32);
            crypto.getRandomValues(seed);

            // 使用 seed 生成密钥对
            // 注意：这是简化的实现，真实场景应该使用专门的加密库如 noble-ed25519
            const publicKeyBase64 = await this.derivePublicKey(seed);
            const privateKeyBase64 = this.bufferToBase64(seed);

            return {
                publicKey: publicKeyBase64,
                privateKey: privateKeyBase64,
                seed: privateKeyBase64
            };
        } catch (error) {
            console.error('Failed to generate key pair:', error);
            throw error;
        }
    }

    /**
     * 从种子推导公钥（简化的 Ed25519 实现）
     * 在实际生产环境中应该使用 noble-ed25519 或类似库
     * @param {Uint8Array} seed - 32字节的种子
     * @returns {Promise<string>} Base64 编码的公钥
     */
    async derivePublicKey(seed) {
        // 这是一个简化版本，用于演示
        // 真实的 Ed25519 公钥推导需要使用专门的加密库
        // 这里我们使用 SHA-256 哈希作为公钥的替代方案
        const hashBuffer = await crypto.subtle.digest('SHA-256', seed);
        const hashArray = Array.from(new Uint8Array(hashBuffer));
        const hashHex = hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
        return this.bufferToBase64(new Uint8Array(hashBuffer));
    }

    /**
     * 对消息进行签名
     * @param {string} message - 要签名的消息
     * @param {string} privateKey - Base64 编码的私钥
     * @returns {Promise<{signature: string, message: string}>}
     */
    async signMessage(message, privateKey) {
        try {
            const privateKeyBytes = this.base64ToBuffer(privateKey);

            // 使用 HMAC-SHA256 进行签名（简化实现）
            const encoder = new TextEncoder();
            const messageBytes = encoder.encode(message);

            const key = await crypto.subtle.importKey(
                'raw',
                privateKeyBytes,
                { name: 'HMAC', hash: 'SHA-256' },
                false,
                ['sign']
            );

            const signature = await crypto.subtle.sign(
                'HMAC',
                key,
                messageBytes
            );

            return {
                signature: this.bufferToBase64(new Uint8Array(signature)),
                message: message
            };
        } catch (error) {
            console.error('Failed to sign message:', error);
            throw error;
        }
    }

    /**
     * 验证签名
     * @param {string} message - 原始消息
     * @param {string} signature - Base64 编码的签名
     * @param {string} publicKey - Base64 编码的公钥
     * @returns {Promise<boolean>}
     */
    async verifySignature(message, signature, publicKey) {
        try {
            const signatureBytes = this.base64ToBuffer(signature);
            const publicKeyBytes = this.base64ToBuffer(publicKey);

            // 重新计算签名以验证
            const encoder = new TextEncoder();
            const messageBytes = encoder.encode(message);

            const key = await crypto.subtle.importKey(
                'raw',
                publicKeyBytes,
                { name: 'HMAC', hash: 'SHA-256' },
                false,
                ['verify']
            );

            const isValid = await crypto.subtle.verify(
                'HMAC',
                key,
                signatureBytes,
                messageBytes
            );

            return isValid;
        } catch (error) {
            console.error('Failed to verify signature:', error);
            return false;
        }
    }

    /**
     * 加密数据（用于存储私钥）
     * @param {string} data - 要加密的数据
     * @param {string} password - 加密密码
     * @returns {Promise<{encrypted: string, salt: string, iv: string}>}
     */
    async encryptData(data, password) {
        try {
            const encoder = new TextEncoder();
            const dataBytes = encoder.encode(data);
            const passwordBytes = encoder.encode(password);

            // 生成随机 salt 和 IV
            const salt = crypto.getRandomValues(new Uint8Array(16));
            const iv = crypto.getRandomValues(new Uint8Array(12));

            // 从密码派生密钥
            const key = await crypto.subtle.importKey(
                'raw',
                passwordBytes,
                'PBKDF2',
                false,
                ['deriveKey']
            );

            const derivedKey = await crypto.subtle.deriveKey(
                {
                    name: 'PBKDF2',
                    salt: salt,
                    iterations: 100000,
                    hash: 'SHA-256'
                },
                key,
                { name: 'AES-GCM', length: 256 },
                false,
                ['encrypt']
            );

            // 加密数据
            const encrypted = await crypto.subtle.encrypt(
                { name: 'AES-GCM', iv: iv },
                derivedKey,
                dataBytes
            );

            return {
                encrypted: this.bufferToBase64(new Uint8Array(encrypted)),
                salt: this.bufferToBase64(salt),
                iv: this.bufferToBase64(iv)
            };
        } catch (error) {
            console.error('Failed to encrypt data:', error);
            throw error;
        }
    }

    /**
     * 解密数据
     * @param {string} encryptedData - Base64 编码的加密数据
     * @param {string} password - 解密密码
     * @param {string} salt - Base64 编码的 salt
     * @param {string} iv - Base64 编码的 iv
     * @returns {Promise<string>}
     */
    async decryptData(encryptedData, password, salt, iv) {
        try {
            const encoder = new TextEncoder();
            const decoder = new TextDecoder();

            const encryptedBytes = this.base64ToBuffer(encryptedData);
            const passwordBytes = encoder.encode(password);
            const saltBytes = this.base64ToBuffer(salt);
            const ivBytes = this.base64ToBuffer(iv);

            // 从密码派生密钥
            const key = await crypto.subtle.importKey(
                'raw',
                passwordBytes,
                'PBKDF2',
                false,
                ['deriveKey']
            );

            const derivedKey = await crypto.subtle.deriveKey(
                {
                    name: 'PBKDF2',
                    salt: saltBytes,
                    iterations: 100000,
                    hash: 'SHA-256'
                },
                key,
                { name: 'AES-GCM', length: 256 },
                false,
                ['decrypt']
            );

            // 解密数据
            const decrypted = await crypto.subtle.decrypt(
                { name: 'AES-GCM', iv: ivBytes },
                derivedKey,
                encryptedBytes
            );

            return decoder.decode(decrypted);
        } catch (error) {
            console.error('Failed to decrypt data:', error);
            throw error;
        }
    }

    /**
     * 生成 DID
     * @param {string} gameId - 游戏 ID
     * @param {string} playerId - 玩家 ID
     * @returns {string} DID 字符串
     */
    generateDID(gameId, playerId) {
        return `did:player:${gameId}:${playerId}`;
    }

    /**
     * 构造 DID 文档
     * @param {string} did - DID 字符串
     * @param {string} publicKey - Base64 编码的公钥
     * @param {string} gameId - 游戏 ID
     * @param {string} playerId - 玩家 ID
     * @returns {Object} DID 文档
     */
    createDIDDocument(did, publicKey, gameId, playerId) {
        return {
            '@context': [
                'https://www.w3.org/ns/did/v1',
                'https://game.example.com/contexts/player/v1'
            ],
            id: did,
            verificationMethod: [{
                id: `${did}#key-1`,
                type: 'Ed25519VerificationKey2018',
                controller: did,
                publicKeyHex: publicKey
            }],
            service: [{
                id: `${did}#game-service`,
                type: 'GameService',
                serviceEndpoint: {
                    gameEndpoint: `https://game.example.com/players/${playerId}`,
                    gameID: gameId,
                    playerID: playerId
                }
            }],
            created: new Date().toISOString()
        };
    }

    /**
     * 辅助方法：Buffer 转 Base64
     * @param {Uint8Array} buffer
     * @returns {string}
     */
    bufferToBase64(buffer) {
        const binary = String.fromCharCode(...buffer);
        return btoa(binary);
    }

    /**
     * 辅助方法：Base64 转 Buffer
     * @param {string} base64
     * @returns {Uint8Array}
     */
    base64ToBuffer(base64) {
        const binary = atob(base64);
        const bytes = new Uint8Array(binary.length);
        for (let i = 0; i < binary.length; i++) {
            bytes[i] = binary.charCodeAt(i);
        }
        return bytes;
    }

    /**
     * 生成 UUID v4
     * @returns {string}
     */
    generateUUID() {
        return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
            const r = Math.random() * 16 | 0;
            const v = c === 'x' ? r : (r & 0x3 | 0x8);
            return v.toString(16);
        });
    }
}

// 全局实例
const cryptoUtils = new CryptoUtils();
