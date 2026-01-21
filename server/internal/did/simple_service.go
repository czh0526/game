package did

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/czh0526/game/server/pkg/did"
)

// SimpleService 简化的DID服务
type SimpleService struct {
	dids  map[string]*did.SimpleDID
	mutex sync.RWMutex
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
	DID        string               `json:"did"`
	DIDDoc     *did.DIDDocument     `json:"didDocument"`
	PrivateKey string               `json:"privateKey"`
	PublicKey  string               `json:"publicKey"`
}

// ResolveDIDResponse 解析DID响应
type ResolveDIDResponse struct {
	DID    string           `json:"did"`
	DIDDoc *did.DIDDocument `json:"didDocument"`
}

// NewSimpleService 创建新的简化DID服务
func NewSimpleService() *SimpleService {
	return &SimpleService{
		dids: make(map[string]*did.SimpleDID),
	}
}

// HandleCreateDID 处理创建DID请求
func (s *SimpleService) HandleCreateDID(w http.ResponseWriter, r *http.Request) {
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

	if req.PlayerID == "" {
		req.PlayerID = fmt.Sprintf("player_%d", len(s.dids)+1)
	}

	// 创建DID
	playerDID, err := did.CreatePlayerDID(req.GameID, req.PlayerID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create DID: %v", err), http.StatusInternalServerError)
		return
	}

	// 存储DID
	s.mutex.Lock()
	s.dids[playerDID.ID] = playerDID
	s.mutex.Unlock()

	// 构建响应
	response := CreateDIDResponse{
		DID:        playerDID.ID,
		DIDDoc:     playerDID.ToDIDDocument(),
		PrivateKey: playerDID.PrivateKey,
		PublicKey:  playerDID.PublicKey,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleResolveDID 处理解析DID请求
func (s *SimpleService) HandleResolveDID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	didID := r.URL.Query().Get("did")
	if didID == "" {
		http.Error(w, "did parameter is required", http.StatusBadRequest)
		return
	}

	// 查找DID
	s.mutex.RLock()
	playerDID, exists := s.dids[didID]
	s.mutex.RUnlock()

	if !exists {
		http.Error(w, "DID not found", http.StatusNotFound)
		return
	}

	// 构建响应
	response := ResolveDIDResponse{
		DID:    didID,
		DIDDoc: playerDID.ToDIDDocument(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CreatePlayerDID 程序化创建玩家DID
func (s *SimpleService) CreatePlayerDID(gameID, playerID, nickname string, level int) (*CreateDIDResponse, error) {
	playerDID, err := did.CreatePlayerDID(gameID, playerID)
	if err != nil {
		return nil, fmt.Errorf("create DID: %w", err)
	}

	// 存储DID
	s.mutex.Lock()
	s.dids[playerDID.ID] = playerDID
	s.mutex.Unlock()

	return &CreateDIDResponse{
		DID:        playerDID.ID,
		DIDDoc:     playerDID.ToDIDDocument(),
		PrivateKey: playerDID.PrivateKey,
		PublicKey:  playerDID.PublicKey,
	}, nil
}

// ResolveDID 程序化解析DID
func (s *SimpleService) ResolveDID(didID string) (*ResolveDIDResponse, error) {
	s.mutex.RLock()
	playerDID, exists := s.dids[didID]
	s.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("DID not found: %s", didID)
	}

	return &ResolveDIDResponse{
		DID:    didID,
		DIDDoc: playerDID.ToDIDDocument(),
	}, nil
}

// GetDID 获取DID
func (s *SimpleService) GetDID(didID string) (*did.SimpleDID, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	playerDID, exists := s.dids[didID]
	if !exists {
		return nil, fmt.Errorf("DID not found: %s", didID)
	}

	return playerDID, nil
}