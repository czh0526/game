package did

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// SimpleDID 简化的DID实现
type SimpleDID struct {
	ID         string    `json:"id"`
	PublicKey  string    `json:"publicKey"`
	PrivateKey string    `json:"privateKey,omitempty"`
	GameID     string    `json:"gameId"`
	PlayerID   string    `json:"playerId"`
	CreatedAt  time.Time `json:"createdAt"`
}

// DIDDocument DID文档
type DIDDocument struct {
	Context            []string                   `json:"@context"`
	ID                 string                     `json:"id"`
	VerificationMethod []VerificationMethod       `json:"verificationMethod"`
	Service            []Service                  `json:"service"`
	CreatedAt          time.Time                  `json:"created"`
}

// VerificationMethod 验证方法
type VerificationMethod struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Controller string `json:"controller"`
	PublicKey  string `json:"publicKeyHex"`
}

// Service 服务端点
type Service struct {
	ID              string                 `json:"id"`
	Type            string                 `json:"type"`
	ServiceEndpoint map[string]interface{} `json:"serviceEndpoint"`
}

// CreatePlayerDID 创建玩家DID
func CreatePlayerDID(gameID, playerID string) (*SimpleDID, error) {
	// 生成密钥对
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate key pair: %w", err)
	}

	// 构建DID
	did := &SimpleDID{
		ID:         fmt.Sprintf("did:player:%s:%s", gameID, playerID),
		PublicKey:  hex.EncodeToString(publicKey),
		PrivateKey: hex.EncodeToString(privateKey),
		GameID:     gameID,
		PlayerID:   playerID,
		CreatedAt:  time.Now(),
	}

	return did, nil
}

// ToDIDDocument 转换为DID文档
func (d *SimpleDID) ToDIDDocument() *DIDDocument {
	return &DIDDocument{
		Context: []string{
			"https://www.w3.org/ns/did/v1",
			"https://game.example.com/contexts/player/v1",
		},
		ID: d.ID,
		VerificationMethod: []VerificationMethod{
			{
				ID:         d.ID + "#key-1",
				Type:       "Ed25519VerificationKey2018",
				Controller: d.ID,
				PublicKey:  d.PublicKey,
			},
		},
		Service: []Service{
			{
				ID:   d.ID + "#game-service",
				Type: "GameService",
				ServiceEndpoint: map[string]interface{}{
					"gameEndpoint": fmt.Sprintf("https://game.example.com/players/%s", d.PlayerID),
					"gameID":       d.GameID,
					"playerID":     d.PlayerID,
				},
			},
		},
		CreatedAt: d.CreatedAt,
	}
}

// Sign 签名消息
func (d *SimpleDID) Sign(message []byte) ([]byte, error) {
	privateKeyBytes, err := hex.DecodeString(d.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("decode private key: %w", err)
	}

	signature := ed25519.Sign(privateKeyBytes, message)
	return signature, nil
}

// Verify 验证签名
func (d *SimpleDID) Verify(message, signature []byte) bool {
	publicKeyBytes, err := hex.DecodeString(d.PublicKey)
	if err != nil {
		return false
	}

	return ed25519.Verify(publicKeyBytes, message, signature)
}

// ToJSON 转换为JSON
func (d *SimpleDID) ToJSON() ([]byte, error) {
	return json.Marshal(d)
}

// FromJSON 从JSON解析
func FromJSON(data []byte) (*SimpleDID, error) {
	var did SimpleDID
	err := json.Unmarshal(data, &did)
	return &did, err
}