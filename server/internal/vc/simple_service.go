package vc

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/czh0526/game/server/internal/did"
	"github.com/czh0526/game/server/pkg/vc"
)

// SimpleService 简化的VC服务
type SimpleService struct {
	didService  *did.SimpleService
	credentials map[string]*vc.SimpleCredential
	issuerDID   string
	mutex       sync.RWMutex
}

// IssueCredentialRequest 颁发凭证请求
type IssueCredentialRequest struct {
	PlayerDID   string                `json:"playerDid"`
	Type        string                `json:"type"`
	Subject     vc.CredentialSubject  `json:"credentialSubject"`
	ExpiresAt   *time.Time            `json:"expiresAt,omitempty"`
}

// IssueCredentialResponse 颁发凭证响应
type IssueCredentialResponse struct {
	Credential *vc.SimpleCredential `json:"credential"`
	ID         string               `json:"id"`
}

// VerifyCredentialRequest 验证凭证请求
type VerifyCredentialRequest struct {
	Credential *vc.SimpleCredential `json:"credential"`
}

// VerifyCredentialResponse 验证凭证响应
type VerifyCredentialResponse struct {
	Valid   bool   `json:"valid"`
	Message string `json:"message,omitempty"`
}

// NewSimpleService 创建新的简化VC服务
func NewSimpleService(didService *did.SimpleService) (*SimpleService, error) {
	// 创建颁发者DID
	issuerResponse, err := didService.CreatePlayerDID("system", "game-server", "Game Server", 999)
	if err != nil {
		return nil, fmt.Errorf("create issuer DID: %w", err)
	}

	return &SimpleService{
		didService:  didService,
		credentials: make(map[string]*vc.SimpleCredential),
		issuerDID:   issuerResponse.DID,
	}, nil
}

// HandleIssueCredential 处理颁发凭证请求
func (s *SimpleService) HandleIssueCredential(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req IssueCredentialRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// 验证请求
	if req.PlayerDID == "" {
		http.Error(w, "playerDid is required", http.StatusBadRequest)
		return
	}
	if req.Type == "" {
		http.Error(w, "type is required", http.StatusBadRequest)
		return
	}

	// 颁发凭证
	credential, err := s.IssueCredential(req.PlayerDID, req.Type, req.Subject, req.ExpiresAt)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to issue credential: %v", err), http.StatusInternalServerError)
		return
	}

	response := IssueCredentialResponse{
		Credential: credential,
		ID:         credential.ID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleVerifyCredential 处理验证凭证请求
func (s *SimpleService) HandleVerifyCredential(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req VerifyCredentialRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	if req.Credential == nil {
		http.Error(w, "credential is required", http.StatusBadRequest)
		return
	}

	// 验证凭证
	valid, message := s.VerifyCredential(req.Credential)

	response := VerifyCredentialResponse{
		Valid:   valid,
		Message: message,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// IssueCredential 颁发凭证
func (s *SimpleService) IssueCredential(playerDID, credType string, subject vc.CredentialSubject, expiresAt *time.Time) (*vc.SimpleCredential, error) {
	// 验证玩家DID是否存在
	_, err := s.didService.ResolveDID(playerDID)
	if err != nil {
		return nil, fmt.Errorf("invalid player DID: %w", err)
	}

	// 颁发凭证
	credential, err := vc.IssueCredential(s.issuerDID, playerDID, credType, subject)
	if err != nil {
		return nil, fmt.Errorf("issue credential: %w", err)
	}

	// 设置过期时间
	if expiresAt != nil {
		credential.ExpirationDate = expiresAt
	}

	// 存储凭证
	s.mutex.Lock()
	s.credentials[credential.ID] = credential
	s.mutex.Unlock()

	return credential, nil
}

// VerifyCredential 验证凭证
func (s *SimpleService) VerifyCredential(credential *vc.SimpleCredential) (bool, string) {
	// 使用简化的验证逻辑
	valid, message := vc.VerifyCredential(credential, s.issuerDID)
	if !valid {
		return valid, message
	}

	// 检查凭证是否存在于存储中
	s.mutex.RLock()
	_, exists := s.credentials[credential.ID]
	s.mutex.RUnlock()

	if !exists {
		return false, "credential not found in registry"
	}

	return true, "credential is valid"
}

// IssueAchievementCredential 颁发成就凭证的便捷方法
func (s *SimpleService) IssueAchievementCredential(playerDID, gameID, playerID, achievement string, score int) (*vc.SimpleCredential, error) {
	now := time.Now()
	subject := vc.CredentialSubject{
		PlayerID:    playerID,
		GameID:      gameID,
		Achievement: achievement,
		Score:       score,
		CompletedAt: &now,
		Attributes: map[string]interface{}{
			"difficulty": "normal",
			"category":   "achievement",
		},
	}

	return s.IssueCredential(playerDID, "AchievementCredential", subject, nil)
}

// IssueLevelCredential 颁发等级凭证的便捷方法
func (s *SimpleService) IssueLevelCredential(playerDID, gameID, playerID string, level int) (*vc.SimpleCredential, error) {
	subject := vc.CredentialSubject{
		PlayerID: playerID,
		GameID:   gameID,
		Level:    level,
		Attributes: map[string]interface{}{
			"category": "level",
		},
	}

	return s.IssueCredential(playerDID, "LevelCredential", subject, nil)
}