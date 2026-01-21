package did

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hyperledger/aries-framework-go/component/models/did"
	vdrspi "github.com/hyperledger/aries-framework-go/spi/vdr"
	"github.com/hyperledger/aries-framework-go/spi/storage"
)

const (
	// DIDMethod 定义 did:player 方法名
	DIDMethod = "player"
	// StoreNamespace 存储命名空间
	StoreNamespace = "did_player"
)

// PlayerVDR 实现游戏玩家DID方法
type PlayerVDR struct {
	store        storage.Store
	gameRegistry GameRegistry
}

// GameRegistry 游戏注册中心接口
type GameRegistry interface {
	ValidatePlayer(gameID, playerID string) error
	GetPlayerInfo(gameID, playerID string) (*PlayerInfo, error)
	RegisterPlayer(gameID, playerID string, playerInfo *PlayerInfo) error
}

// PlayerInfo 玩家信息
type PlayerInfo struct {
	PlayerID   string                 `json:"playerId"`
	GameID     string                 `json:"gameId"`
	Nickname   string                 `json:"nickname"`
	Level      int                    `json:"level"`
	Attributes map[string]interface{} `json:"attributes"`
	CreatedAt  time.Time              `json:"createdAt"`
	UpdatedAt  time.Time              `json:"updatedAt"`
}

// NewPlayerVDR 创建新的PlayerVDR实例
func NewPlayerVDR(storageProvider storage.Provider, gameRegistry GameRegistry) (*PlayerVDR, error) {
	store, err := storageProvider.OpenStore(StoreNamespace)
	if err != nil {
		return nil, fmt.Errorf("open player store: %w", err)
	}

	return &PlayerVDR{
		store:        store,
		gameRegistry: gameRegistry,
	}, nil
}

// Accept 判断是否接受did:player方法
func (v *PlayerVDR) Accept(method string, opts ...vdrspi.DIDMethodOption) bool {
	return method == DIDMethod
}

// Create 创建新的玩家DID
func (v *PlayerVDR) Create(didDoc *did.Doc, opts ...vdrspi.DIDMethodOption) (*did.DocResolution, error) {
	docOpts := &vdrspi.DIDMethodOpts{Values: make(map[string]interface{})}
	for _, opt := range opts {
		opt(docOpts)
	}

	// 获取游戏相关参数
	gameID, ok := docOpts.Values["gameID"].(string)
	if !ok {
		return nil, fmt.Errorf("gameID is required for did:player")
	}

	playerID, ok := docOpts.Values["playerID"].(string)
	if !ok {
		// 如果没有提供playerID，生成一个新的
		playerID = uuid.New().String()
	}

	// 构建did:player DID
	playerDID := fmt.Sprintf("did:player:%s:%s", gameID, playerID)

	// 验证游戏和玩家的有效性
	if err := v.gameRegistry.ValidatePlayer(gameID, playerID); err != nil {
		return nil, fmt.Errorf("invalid player: %w", err)
	}

	// 构建DID文档
	didDoc.ID = playerDID
	didDoc.Context = []string{
		"https://www.w3.org/ns/did/v1",
		"https://game.example.com/contexts/player/v1",
	}

	// 添加创建时间
	now := time.Now()
	didDoc.Created = &now

	// 更新验证方法的ID
	for i := range didDoc.VerificationMethod {
		didDoc.VerificationMethod[i].ID = fmt.Sprintf("%s#key-%d", playerDID, i+1)
		didDoc.VerificationMethod[i].Controller = playerDID
	}

	// 添加游戏特定的服务端点
	gameService := did.Service{
		ID:   fmt.Sprintf("%s#game-service", playerDID),
		Type: "GameService",
		ServiceEndpoint: map[string]interface{}{
			"gameEndpoint": fmt.Sprintf("https://game.example.com/players/%s", playerID),
			"gameID":       gameID,
			"playerID":     playerID,
		},
	}
	didDoc.Service = append(didDoc.Service, gameService)

	// DIDComm消息服务端点
	didCommService := did.Service{
		ID:   fmt.Sprintf("%s#didcomm", playerDID),
		Type: "DIDCommMessaging",
		ServiceEndpoint: map[string]interface{}{
			"uri":    fmt.Sprintf("https://game.example.com/didcomm/%s", playerID),
			"accept": []string{"didcomm/v2"},
		},
	}
	didDoc.Service = append(didDoc.Service, didCommService)

	// 存储DID文档
	docBytes, err := didDoc.JSONBytes()
	if err != nil {
		return nil, fmt.Errorf("marshal did doc: %w", err)
	}

	if err := v.store.Put(playerDID, docBytes); err != nil {
		return nil, fmt.Errorf("store did doc: %w", err)
	}

	return &did.DocResolution{
		Context:     []string{"https://w3id.org/did-resolution/v1"},
		DIDDocument: didDoc,
	}, nil
}

// Read 解析did:player DID
func (v *PlayerVDR) Read(didID string, opts ...vdrspi.DIDMethodOption) (*did.DocResolution, error) {
	// 解析DID格式: did:player:gameID:playerID
	parts := strings.Split(didID, ":")
	if len(parts) != 4 || parts[0] != "did" || parts[1] != "player" {
		return nil, fmt.Errorf("invalid did:player format: %s", didID)
	}

	gameID, playerID := parts[2], parts[3]

	// 从存储中获取DID文档
	docBytes, err := v.store.Get(didID)
	if err != nil {
		if errors.Is(err, storage.ErrDataNotFound) {
			// 如果本地没有，尝试从游戏注册中心获取
			return v.resolveFromGameRegistry(gameID, playerID, didID)
		}
		return nil, fmt.Errorf("get did doc: %w", err)
	}

	// 解析DID文档
	didDoc, err := did.ParseDocument(docBytes)
	if err != nil {
		return nil, fmt.Errorf("parse did doc: %w", err)
	}

	return &did.DocResolution{
		Context:     []string{"https://w3id.org/did-resolution/v1"},
		DIDDocument: didDoc,
	}, nil
}

// Update 更新玩家DID文档
func (v *PlayerVDR) Update(didDoc *did.Doc, opts ...vdrspi.DIDMethodOption) error {
	// 验证DID格式
	if !strings.HasPrefix(didDoc.ID, "did:player:") {
		return fmt.Errorf("not a did:player DID: %s", didDoc.ID)
	}

	// 验证更新权限
	if err := v.validateUpdatePermission(didDoc, opts...); err != nil {
		return fmt.Errorf("update permission denied: %w", err)
	}

	// 更新时间戳
	now := time.Now()
	didDoc.Updated = &now

	// 存储更新后的文档
	docBytes, err := didDoc.JSONBytes()
	if err != nil {
		return fmt.Errorf("marshal updated did doc: %w", err)
	}

	return v.store.Put(didDoc.ID, docBytes)
}

// Deactivate 停用玩家DID
func (v *PlayerVDR) Deactivate(didID string, opts ...vdrspi.DIDMethodOption) error {
	// 验证权限
	if err := v.validateDeactivatePermission(didID, opts...); err != nil {
		return fmt.Errorf("deactivate permission denied: %w", err)
	}

	// 标记为已停用而不是删除
	deactivatedDoc := &did.Doc{
		ID:      didID,
		Context: []string{"https://www.w3.org/ns/did/v1"},
		// 空的验证方法表示已停用
		VerificationMethod: []did.VerificationMethod{},
	}

	docBytes, err := deactivatedDoc.JSONBytes()
	if err != nil {
		return fmt.Errorf("marshal deactivated did doc: %w", err)
	}

	return v.store.Put(didID, docBytes)
}

// Close 关闭VDR
func (v *PlayerVDR) Close() error {
	return nil
}

// resolveFromGameRegistry 从游戏注册中心解析DID
func (v *PlayerVDR) resolveFromGameRegistry(gameID, playerID, didID string) (*did.DocResolution, error) {
	playerInfo, err := v.gameRegistry.GetPlayerInfo(gameID, playerID)
	if err != nil {
		return nil, fmt.Errorf("player not found in game registry: %w", err)
	}

	// 基于玩家信息构建基础DID文档
	didDoc := &did.Doc{
		ID:      didID,
		Context: []string{"https://www.w3.org/ns/did/v1", "https://game.example.com/contexts/player/v1"},
		Service: []did.Service{
			{
				ID:   fmt.Sprintf("%s#game-service", didID),
				Type: "GameService",
				ServiceEndpoint: map[string]interface{}{
					"gameEndpoint": fmt.Sprintf("https://game.example.com/players/%s", playerID),
					"gameID":       gameID,
					"playerID":     playerID,
					"nickname":     playerInfo.Nickname,
					"level":        playerInfo.Level,
				},
			},
		},
	}

	// 缓存到本地存储
	docBytes, err := didDoc.JSONBytes()
	if err == nil {
		v.store.Put(didID, docBytes)
	}

	return &did.DocResolution{
		Context:     []string{"https://w3id.org/did-resolution/v1"},
		DIDDocument: didDoc,
	}, nil
}

// validateUpdatePermission 验证更新权限
func (v *PlayerVDR) validateUpdatePermission(didDoc *did.Doc, opts ...vdrspi.DIDMethodOption) error {
	// 这里可以实现更复杂的权限验证逻辑
	// 例如验证签名、检查控制器权限等
	return nil
}

// validateDeactivatePermission 验证停用权限
func (v *PlayerVDR) validateDeactivatePermission(didID string, opts ...vdrspi.DIDMethodOption) error {
	// 这里可以实现更复杂的权限验证逻辑
	return nil
}

// PlayerDIDOptions 创建DID时的选项
type PlayerDIDOptions struct {
	GameID   string
	PlayerID string
	Nickname string
	Level    int
}

// WithGameID 设置游戏ID选项
func WithGameID(gameID string) vdrspi.DIDMethodOption {
	return func(opts *vdrspi.DIDMethodOpts) {
		opts.Values["gameID"] = gameID
	}
}

// WithPlayerID 设置玩家ID选项
func WithPlayerID(playerID string) vdrspi.DIDMethodOption {
	return func(opts *vdrspi.DIDMethodOpts) {
		opts.Values["playerID"] = playerID
	}
}

// WithNickname 设置昵称选项
func WithNickname(nickname string) vdrspi.DIDMethodOption {
	return func(opts *vdrspi.DIDMethodOpts) {
		opts.Values["nickname"] = nickname
	}
}

// WithLevel 设置等级选项
func WithLevel(level int) vdrspi.DIDMethodOption {
	return func(opts *vdrspi.DIDMethodOpts) {
		opts.Values["level"] = level
	}
}