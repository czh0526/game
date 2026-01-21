package vc

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/hyperledger/aries-framework-go/pkg/doc/verifiable"
	"github.com/hyperledger/aries-framework-go/spi/storage"
	"github.com/hyperledger/aries-framework-go/component/storageutil/mem"
	
	"github.com/czh0526/game/server/internal/did"
)

// Service VC服务
type Service struct {
	store      storage.Store
	didService *did.Service
	issuerDID  string
	issuerKey  ed25519.PrivateKey
}

// GameCredentialSubject 游戏凭证主体
type GameCredentialSubject struct {
	ID           string                 `json:"id"`
	PlayerID     string                 `json:"playerId"`
	GameID       string                 `json:"gameId"`
	Achievement  string                 `json:"achievement,omitempty"`
	Level        int                    `json:"level,omitempty"`
	Score        int                    `json:"score,omitempty"`
	Skills       []string               `json:"skills,omitempty"`
	Items        []string               `json:"items,omitempty"`
	Attributes   map[string]interface{} `json:"attributes,omitempty"`
	CompletedAt  *time.Time             `json:"completedAt,omitempty"`
}

// IssueCredentialRequest 颁发凭证请求
type IssueCredentialRequest struct {
	PlayerDID   string                    `json:"playerDid"`
	Type        string                    `json:"type"`
	Subject     GameCredentialSubject     `json:"credentialSubject"`
	ExpiresAt   *time.Time                `json:"expiresAt,omitempty"`
}

// IssueCredentialResponse 颁发凭证响应
type IssueCredentialResponse struct {
	Credential *verifiable.Credential `json:"credential"`
	ID         string                 `json:"id"`
}

// VerifyCredentialRequest 验证凭证请求
type VerifyCredentialRequest struct {
	Credential *verifiable.Credential `json:"credential"`
}

// VerifyCredentialResponse 验证凭证响应
type VerifyCredentialResponse struct {
	Valid   bool   `json:"valid"`
	Message string `json:"message,omitempty"`
}

// CredentialType 凭证类型常量
const (
	AchievementCredentialType = "AchievementCredential"
	LevelCredentialType      = "LevelCredential"
	SkillCredentialType      = "SkillCredential"
	ItemCredentialType       = "ItemCredential"
)

// NewService 创建新的VC服务
func NewService(didService *did.Service) (*Service, error) {
	// 创建存储
	storageProvider := mem.NewProvider()
	store, err := storageProvider.OpenStore("vc_service")
	if err != nil {
		return nil, fmt.Errorf("open vc store: %w", err)
	}

	// 创建或获取颁发者DID
	issuerDID, issuerKey, err := createIssuerDID(didService)
	if err != nil {
		return nil, fmt.Errorf("create issuer DID: %w", err)
	}

	return &Service{
		store:      store,
		didService: didService,
		issuerDID:  issuerDID,
		issuerKey:  issuerKey,
	}, nil
}

// createIssuerDID 创建颁发者DID
func createIssuerDID(didService *did.Service) (string, ed25519.PrivateKey, error) {
	// 为游戏服务器创建一个特殊的DID
	response, err := didService.CreatePlayerDID("system", "game-server", "Game Server", 999)
	if err != nil {
		return "", nil, fmt.Errorf("create issuer DID: %w", err)
	}

	// 解析私钥
	privateKeyBytes, err := hex.DecodeString(response.PrivateKey)
	if err != nil {
		return "", nil, fmt.Errorf("decode private key: %w", err)
	}

	return response.DID, ed25519.PrivateKey(privateKeyBytes), nil
}

// HandleIssueCredential 处理颁发凭证请求
func (s *Service) HandleIssueCredential(w http.ResponseWriter, r *http.Request) {
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
func (s *Service) HandleVerifyCredential(w http.ResponseWriter, r *http.Request) {
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
func (s *Service) IssueCredential(playerDID, credType string, subject GameCredentialSubject, expiresAt *time.Time) (*verifiable.Credential, error) {
	// 验证玩家DID是否存在
	_, err := s.didService.ResolveDID(playerDID)
	if err != nil {
		return nil, fmt.Errorf("invalid player DID: %w", err)
	}

	// 生成凭证ID
	credentialID := fmt.Sprintf("urn:uuid:%s", uuid.New().String())

	// 设置凭证主体ID
	subject.ID = playerDID

	// 创建凭证
	now := time.Now()
	credential := &verifiable.Credential{
		Context: []string{
			"https://www.w3.org/2018/credentials/v1",
			"https://game.example.com/contexts/credentials/v1",
		},
		ID: credentialID,
		Type: []string{
			"VerifiableCredential",
			credType,
		},
		Issuer: verifiable.Issuer{
			ID: s.issuerDID,
		},
		IssuanceDate: &now,
		Subject:      subject,
	}

	// 设置过期时间
	if expiresAt != nil {
		credential.ExpirationDate = expiresAt
	}

	// 添加游戏特定的上下文信息
	credential.CustomFields = map[string]interface{}{
		"gameContext": map[string]interface{}{
			"gameId":    subject.GameID,
			"playerId":  subject.PlayerID,
			"issuedBy":  "game-server",
			"timestamp": now.Unix(),
		},
	}

	// 存储凭证
	credentialBytes, err := credential.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("marshal credential: %w", err)
	}

	storageKey := fmt.Sprintf("credential:%s", credentialID)
	if err := s.store.Put(storageKey, credentialBytes); err != nil {
		return nil, fmt.Errorf("store credential: %w", err)
	}

	// 为玩家建立凭证索引
	playerCredKey := fmt.Sprintf("player_credentials:%s", playerDID)
	playerCreds, err := s.getPlayerCredentials(playerDID)
	if err != nil && err != storage.ErrDataNotFound {
		return nil, fmt.Errorf("get player credentials: %w", err)
	}

	playerCreds = append(playerCreds, credentialID)
	playerCredsBytes, err := json.Marshal(playerCreds)
	if err != nil {
		return nil, fmt.Errorf("marshal player credentials: %w", err)
	}

	if err := s.store.Put(playerCredKey, playerCredsBytes); err != nil {
		return nil, fmt.Errorf("store player credentials index: %w", err)
	}

	return credential, nil
}

// VerifyCredential 验证凭证
func (s *Service) VerifyCredential(credential *verifiable.Credential) (bool, string) {
	// 基础验证
	if credential == nil {
		return false, "credential is nil"
	}

	// 检查颁发者
	if credential.Issuer.ID != s.issuerDID {
		return false, "invalid issuer"
	}

	// 检查过期时间
	if credential.ExpirationDate != nil && credential.ExpirationDate.Before(time.Now()) {
		return false, "credential has expired"
	}

	// 检查颁发时间
	if credential.IssuanceDate != nil && credential.IssuanceDate.After(time.Now()) {
		return false, "credential issued in the future"
	}

	// 验证凭证是否存在于存储中
	storageKey := fmt.Sprintf("credential:%s", credential.ID)
	_, err := s.store.Get(storageKey)
	if err != nil {
		if err == storage.ErrDataNotFound {
			return false, "credential not found in registry"
		}
		return false, fmt.Sprintf("storage error: %v", err)
	}

	// 验证凭证主体
	if err := s.validateCredentialSubject(credential); err != nil {
		return false, fmt.Sprintf("invalid credential subject: %v", err)
	}

	return true, "credential is valid"
}

// validateCredentialSubject 验证凭证主体
func (s *Service) validateCredentialSubject(credential *verifiable.Credential) error {
	// 这里可以添加更多的业务逻辑验证
	// 例如验证玩家是否真的完成了相应的成就等

	// 基础验证：检查主体是否为有效的玩家DID
	if len(credential.Subject) == 0 {
		return fmt.Errorf("no credential subject")
	}

	// 获取第一个主体（通常游戏凭证只有一个主体）
	subject := credential.Subject[0]
	
	// 检查主体ID是否为有效的DID
	subjectMap, ok := subject.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid subject format")
	}

	subjectID, ok := subjectMap["id"].(string)
	if !ok || subjectID == "" {
		return fmt.Errorf("missing subject id")
	}

	// 验证DID是否存在
	_, err := s.didService.ResolveDID(subjectID)
	if err != nil {
		return fmt.Errorf("invalid subject DID: %w", err)
	}

	return nil
}

// GetPlayerCredentials 获取玩家的所有凭证
func (s *Service) GetPlayerCredentials(playerDID string) ([]*verifiable.Credential, error) {
	credentialIDs, err := s.getPlayerCredentials(playerDID)
	if err != nil {
		return nil, fmt.Errorf("get player credential IDs: %w", err)
	}

	var credentials []*verifiable.Credential
	for _, credID := range credentialIDs {
		storageKey := fmt.Sprintf("credential:%s", credID)
		credBytes, err := s.store.Get(storageKey)
		if err != nil {
			continue // 跳过不存在的凭证
		}

		var credential verifiable.Credential
		if err := json.Unmarshal(credBytes, &credential); err != nil {
			continue // 跳过无法解析的凭证
		}

		credentials = append(credentials, &credential)
	}

	return credentials, nil
}

// getPlayerCredentials 获取玩家凭证ID列表
func (s *Service) getPlayerCredentials(playerDID string) ([]string, error) {
	playerCredKey := fmt.Sprintf("player_credentials:%s", playerDID)
	credBytes, err := s.store.Get(playerCredKey)
	if err != nil {
		if err == storage.ErrDataNotFound {
			return []string{}, nil
		}
		return nil, err
	}

	var credentialIDs []string
	if err := json.Unmarshal(credBytes, &credentialIDs); err != nil {
		return nil, fmt.Errorf("unmarshal credential IDs: %w", err)
	}

	return credentialIDs, nil
}

// RevokeCredential 撤销凭证
func (s *Service) RevokeCredential(credentialID string) error {
	// 标记凭证为已撤销而不是删除
	revokedKey := fmt.Sprintf("revoked:%s", credentialID)
	revokedData := map[string]interface{}{
		"revokedAt": time.Now(),
		"reason":    "revoked by issuer",
	}

	revokedBytes, err := json.Marshal(revokedData)
	if err != nil {
		return fmt.Errorf("marshal revoked data: %w", err)
	}

	return s.store.Put(revokedKey, revokedBytes)
}

// IsCredentialRevoked 检查凭证是否已撤销
func (s *Service) IsCredentialRevoked(credentialID string) (bool, error) {
	revokedKey := fmt.Sprintf("revoked:%s", credentialID)
	_, err := s.store.Get(revokedKey)
	if err != nil {
		if err == storage.ErrDataNotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// IssueAchievementCredential 颁发成就凭证的便捷方法
func (s *Service) IssueAchievementCredential(playerDID, gameID, playerID, achievement string, score int) (*verifiable.Credential, error) {
	now := time.Now()
	subject := GameCredentialSubject{
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

	return s.IssueCredential(playerDID, AchievementCredentialType, subject, nil)
}

// IssueLevelCredential 颁发等级凭证的便捷方法
func (s *Service) IssueLevelCredential(playerDID, gameID, playerID string, level int) (*verifiable.Credential, error) {
	subject := GameCredentialSubject{
		PlayerID: playerID,
		GameID:   gameID,
		Level:    level,
		Attributes: map[string]interface{}{
			"category": "level",
		},
	}

	return s.IssueCredential(playerDID, LevelCredentialType, subject, nil)
}