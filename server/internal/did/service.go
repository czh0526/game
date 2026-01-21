package did

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hyperledger/aries-framework-go/component/models/did"
	"github.com/hyperledger/aries-framework-go/component/vdr"
	"github.com/hyperledger/aries-framework-go/component/vdr/key"
	"github.com/hyperledger/aries-framework-go/component/vdr/web"
	"github.com/hyperledger/aries-framework-go/spi/storage"
	"github.com/hyperledger/aries-framework-go/component/storageutil/mem"
)

// Service DID服务
type Service struct {
	vdrRegistry  *vdr.Registry
	gameRegistry *InMemoryGameRegistry
	playerVDR    *PlayerVDR
}

// CreateDIDRequest 创建DID请求
type CreateDIDRequest struct {
	GameID   string `json:"gameId"`
	PlayerID string `json:"playerId,omitempty"`
	Nickname string `json:"nickname,omitempty"`
	Level    int    `json:"level,omitempty"`
}

// CreateDIDResponse 创建DID响应
type CreateDIDResponse struct {
	DID        string      `json:"did"`
	DIDDoc     *did.Doc    `json:"didDocument"`
	PrivateKey string      `json:"privateKey"`
	PublicKey  string      `json:"publicKey"`
}

// ResolveDIDRequest 解析DID请求
type ResolveDIDRequest struct {
	DID string `json:"did"`
}

// ResolveDIDResponse 解析DID响应
type ResolveDIDResponse struct {
	DID    string   `json:"did"`
	DIDDoc *did.Doc `json:"didDocument"`
}

// NewService 创建新的DID服务
func NewService() (*Service, error) {
	// 创建内存存储提供者
	storageProvider := mem.NewProvider()

	// 创建游戏注册中心
	gameRegistry, err := NewInMemoryGameRegistry(storageProvider)
	if err != nil {
		return nil, fmt.Errorf("create game registry: %w", err)
	}

	// 创建Player VDR
	playerVDR, err := NewPlayerVDR(storageProvider, gameRegistry)
	if err != nil {
		return nil, fmt.Errorf("create player VDR: %w", err)
	}

	// 创建VDR注册中心
	vdrRegistry := vdr.New(
		vdr.WithVDR(key.New()),
		vdr.WithVDR(web.New()),
		vdr.WithVDR(playerVDR),
	)

	return &Service{
		vdrRegistry:  vdrRegistry,
		gameRegistry: gameRegistry,
		playerVDR:    playerVDR,
	}, nil
}

// HandleCreateDID 处理创建DID请求
func (s *Service) HandleCreateDID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateDIDRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// 验证必需字段
	if req.GameID == "" {
		http.Error(w, "gameId is required", http.StatusBadRequest)
		return
	}

	// 生成密钥对
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate key pair: %v", err), http.StatusInternalServerError)
		return
	}

	// 创建基础DID文档
	didDoc := &did.Doc{
		VerificationMethod: []did.VerificationMethod{
			{
				Type:  "Ed25519VerificationKey2018",
				Value: publicKey,
			},
		},
	}

	// 创建DID选项
	opts := []vdr.DIDMethodOption{
		WithGameID(req.GameID),
	}
	if req.PlayerID != "" {
		opts = append(opts, WithPlayerID(req.PlayerID))
	}
	if req.Nickname != "" {
		opts = append(opts, WithNickname(req.Nickname))
	}
	if req.Level > 0 {
		opts = append(opts, WithLevel(req.Level))
	}

	// 创建DID
	docResolution, err := s.vdrRegistry.Create(DIDMethod, didDoc, opts...)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create DID: %v", err), http.StatusInternalServerError)
		return
	}

	// 构建响应
	response := CreateDIDResponse{
		DID:        docResolution.DIDDocument.ID,
		DIDDoc:     docResolution.DIDDocument,
		PrivateKey: fmt.Sprintf("%x", privateKey),
		PublicKey:  fmt.Sprintf("%x", publicKey),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleResolveDID 处理解析DID请求
func (s *Service) HandleResolveDID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	didID := r.URL.Query().Get("did")
	if didID == "" {
		http.Error(w, "did parameter is required", http.StatusBadRequest)
		return
	}

	// 解析DID
	docResolution, err := s.vdrRegistry.Resolve(didID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to resolve DID: %v", err), http.StatusNotFound)
		return
	}

	// 构建响应
	response := ResolveDIDResponse{
		DID:    didID,
		DIDDoc: docResolution.DIDDocument,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetGameRegistry 获取游戏注册中心
func (s *Service) GetGameRegistry() *InMemoryGameRegistry {
	return s.gameRegistry
}

// GetVDRRegistry 获取VDR注册中心
func (s *Service) GetVDRRegistry() *vdr.Registry {
	return s.vdrRegistry
}

// CreatePlayerDID 程序化创建玩家DID
func (s *Service) CreatePlayerDID(gameID, playerID, nickname string, level int) (*CreateDIDResponse, error) {
	// 生成密钥对
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate key pair: %w", err)
	}

	// 创建基础DID文档
	didDoc := &did.Doc{
		VerificationMethod: []did.VerificationMethod{
			{
				Type:  "Ed25519VerificationKey2018",
				Value: publicKey,
			},
		},
	}

	// 创建DID选项
	opts := []vdr.DIDMethodOption{
		WithGameID(gameID),
		WithPlayerID(playerID),
		WithNickname(nickname),
		WithLevel(level),
	}

	// 创建DID
	docResolution, err := s.vdrRegistry.Create(DIDMethod, didDoc, opts...)
	if err != nil {
		return nil, fmt.Errorf("create DID: %w", err)
	}

	return &CreateDIDResponse{
		DID:        docResolution.DIDDocument.ID,
		DIDDoc:     docResolution.DIDDocument,
		PrivateKey: fmt.Sprintf("%x", privateKey),
		PublicKey:  fmt.Sprintf("%x", publicKey),
	}, nil
}

// ResolveDID 程序化解析DID
func (s *Service) ResolveDID(didID string) (*ResolveDIDResponse, error) {
	docResolution, err := s.vdrRegistry.Resolve(didID)
	if err != nil {
		return nil, fmt.Errorf("resolve DID: %w", err)
	}

	return &ResolveDIDResponse{
		DID:    didID,
		DIDDoc: docResolution.DIDDocument,
	}, nil
}