package did

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/czh0526/game/server/internal/aries"
	"github.com/czh0526/game/server/pkg/did"
)

// SimpleService 简化的DID服务
type SimpleService struct {
	dids      map[string]*did.SimpleDID
	mutex     sync.RWMutex
	ariesSvc  *aries.AriesService
	useAries  bool
}

// RegisterDIDRequest 注册DID请求（客户端已生成密钥对）
type RegisterDIDRequest struct {
	DID         string           `json:"did"`
	DIDDoc     *did.DIDDocument `json:"didDocument"`
	PublicKey  string           `json:"publicKey"`
	GameID     string           `json:"gameId"`
	PlayerID   string           `json:"playerId"`
	Nickname   string           `json:"nickname,omitempty"`
	Level      int              `json:"level,omitempty"`
}

// RegisterDIDResponse 注册DID响应
type RegisterDIDResponse struct {
	Success   bool   `json:"success"`
	DID       string `json:"did"`
	Message   string `json:"message,omitempty"`
	Timestamp string `json:"timestamp"`
}

// ResolveDIDResponse 解析DID响应
type ResolveDIDResponse struct {
	DID    string           `json:"did"`
	DIDDoc *did.DIDDocument `json:"didDocument"`
}

// NewSimpleService 创建新的简化DID服务
func NewSimpleService() *SimpleService {
	return &SimpleService{
		dids:     make(map[string]*did.SimpleDID),
		useAries: false,
	}
}

// NewSimpleServiceWithAries 创建使用 Aries 框架的 DID 服务
func NewSimpleServiceWithAries(ariesSvc *aries.AriesService) *SimpleService {
	return &SimpleService{
		dids:     make(map[string]*did.SimpleDID),
		ariesSvc: ariesSvc,
		useAries: true,
	}
}

// HandleRegisterDID 处理注册DID请求（客户端已生成密钥对）
func (s *SimpleService) HandleRegisterDID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RegisterDIDRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// 验证必需字段
	if req.DID == "" {
		http.Error(w, "did is required", http.StatusBadRequest)
		return
	}
	if req.PublicKey == "" {
		http.Error(w, "publicKey is required", http.StatusBadRequest)
		return
	}
	if req.GameID == "" {
		http.Error(w, "gameId is required", http.StatusBadRequest)
		return
	}

	// 验证 DID 格式
	if !did.IsValidPlayerDID(req.DID) {
		http.Error(w, "invalid DID format", http.StatusBadRequest)
		return
	}

	// 验证 DID 文档中的公钥是否匹配
	if req.DIDDoc == nil {
		http.Error(w, "didDocument is required", http.StatusBadRequest)
		return
	}

	if len(req.DIDDoc.VerificationMethod) == 0 {
		http.Error(w, "didDocument must contain verificationMethod", http.StatusBadRequest)
		return
	}

	didPublicKey := req.DIDDoc.VerificationMethod[0].PublicKey
	if didPublicKey != req.PublicKey {
		http.Error(w, "publicKey in request does not match didDocument", http.StatusBadRequest)
		return
	}

	// 检查 DID 是否已存在
	s.mutex.Lock()
	if _, exists := s.dids[req.DID]; exists {
		s.mutex.Unlock()
		http.Error(w, "DID already exists", http.StatusConflict)
		return
	}

	// 创建 SimpleDID 对象（不包含私钥）
	playerDID := &did.SimpleDID{
		ID:        req.DID,
		PublicKey: req.PublicKey,
		GameID:    req.GameID,
		PlayerID:  req.PlayerID,
		CreatedAt: time.Now(),
	}

	// 存储DID
	s.dids[req.DID] = playerDID
	s.mutex.Unlock()

	// 构建响应
	response := RegisterDIDResponse{
		Success:   true,
		DID:       req.DID,
		Message:   "DID registered successfully",
		Timestamp: time.Now().Format(time.RFC3339),
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

// CreateDIDWithAriesRequest 通过Aries创建DID的请求
type CreateDIDWithAriesRequest struct {
	GameID   string `json:"gameId"`
	PlayerID string `json:"playerId"`
	Nickname string `json:"nickname,omitempty"`
	Level    int    `json:"level,omitempty"`
}

// CreateDIDWithAriesResponse 通过Aries创建DID的响应
type CreateDIDWithAriesResponse struct {
	Success    bool   `json:"success"`
	DID        string `json:"did"`
	PublicKey  string `json:"publicKey"`
	PrivateKey string `json:"privateKey"`
	Message    string `json:"message,omitempty"`
	Timestamp  string `json:"timestamp"`
}

// HandleCreateDIDWithAries 处理通过Aries创建DID的请求
func (s *SimpleService) HandleCreateDIDWithAries(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 检查是否启用了Aries服务
	if !s.useAries || s.ariesSvc == nil {
		http.Error(w, "Aries service is not enabled", http.StatusInternalServerError)
		return
	}

	var req CreateDIDWithAriesRequest
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
		http.Error(w, "playerId is required", http.StatusBadRequest)
		return
	}

	// 使用Aries服务创建DID
	ariesResponse, err := s.ariesSvc.CreatePlayerDID(req.GameID, req.PlayerID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create DID with Aries: %v", err), http.StatusInternalServerError)
		return
	}

	// 构建响应
	response := CreateDIDWithAriesResponse{
		Success:    true,
		DID:        ariesResponse.DID,
		PublicKey:  ariesResponse.PublicKey,
		PrivateKey: ariesResponse.PrivateKey,
		Message:    "DID created successfully with Aries framework",
		Timestamp:  time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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