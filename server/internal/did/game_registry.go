package did

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/hyperledger/aries-framework-go/spi/storage"
)

// InMemoryGameRegistry 内存中的游戏注册中心实现
type InMemoryGameRegistry struct {
	players map[string]*PlayerInfo
	games   map[string]*GameInfo
	mutex   sync.RWMutex
	store   storage.Store
}

// GameInfo 游戏信息
type GameInfo struct {
	GameID      string                 `json:"gameId"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	MaxPlayers  int                    `json:"maxPlayers"`
	Status      string                 `json:"status"` // active, inactive, maintenance
	Settings    map[string]interface{} `json:"settings"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
}

// NewInMemoryGameRegistry 创建新的内存游戏注册中心
func NewInMemoryGameRegistry(storageProvider storage.Provider) (*InMemoryGameRegistry, error) {
	store, err := storageProvider.OpenStore("game_registry")
	if err != nil {
		return nil, fmt.Errorf("open game registry store: %w", err)
	}

	registry := &InMemoryGameRegistry{
		players: make(map[string]*PlayerInfo),
		games:   make(map[string]*GameInfo),
		store:   store,
	}

	// 从存储中加载数据
	if err := registry.loadFromStorage(); err != nil {
		return nil, fmt.Errorf("load from storage: %w", err)
	}

	// 如果没有默认游戏，创建一个
	if len(registry.games) == 0 {
		defaultGame := &GameInfo{
			GameID:      "default",
			Name:        "Aries Adventure",
			Description: "Default Aries game world",
			MaxPlayers:  1000,
			Status:      "active",
			Settings: map[string]interface{}{
				"allowGuestPlayers": true,
				"requireInvitation": false,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		registry.games["default"] = defaultGame
		registry.saveToStorage()
	}

	return registry, nil
}

// ValidatePlayer 验证玩家是否有效
func (r *InMemoryGameRegistry) ValidatePlayer(gameID, playerID string) error {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// 检查游戏是否存在且活跃
	game, exists := r.games[gameID]
	if !exists {
		return fmt.Errorf("game %s not found", gameID)
	}

	if game.Status != "active" {
		return fmt.Errorf("game %s is not active (status: %s)", gameID, game.Status)
	}

	// 检查玩家是否已注册
	playerKey := fmt.Sprintf("%s:%s", gameID, playerID)
	player, exists := r.players[playerKey]
	if !exists {
		// 如果游戏允许访客玩家，自动注册
		if allowGuest, ok := game.Settings["allowGuestPlayers"].(bool); ok && allowGuest {
			return r.autoRegisterPlayer(gameID, playerID)
		}
		return fmt.Errorf("player %s not registered in game %s", playerID, gameID)
	}

	// 检查玩家状态
	if status, ok := player.Attributes["status"].(string); ok && status == "banned" {
		return fmt.Errorf("player %s is banned from game %s", playerID, gameID)
	}

	return nil
}

// GetPlayerInfo 获取玩家信息
func (r *InMemoryGameRegistry) GetPlayerInfo(gameID, playerID string) (*PlayerInfo, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	playerKey := fmt.Sprintf("%s:%s", gameID, playerID)
	player, exists := r.players[playerKey]
	if !exists {
		return nil, fmt.Errorf("player %s not found in game %s", playerID, gameID)
	}

	// 返回副本以避免并发修改
	playerCopy := *player
	return &playerCopy, nil
}

// RegisterPlayer 注册玩家
func (r *InMemoryGameRegistry) RegisterPlayer(gameID, playerID string, playerInfo *PlayerInfo) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// 检查游戏是否存在
	game, exists := r.games[gameID]
	if !exists {
		return fmt.Errorf("game %s not found", gameID)
	}

	// 检查玩家数量限制
	currentPlayers := r.countPlayersInGame(gameID)
	if currentPlayers >= game.MaxPlayers {
		return fmt.Errorf("game %s is full (max players: %d)", gameID, game.MaxPlayers)
	}

	playerKey := fmt.Sprintf("%s:%s", gameID, playerID)

	// 设置默认值
	if playerInfo.Attributes == nil {
		playerInfo.Attributes = make(map[string]interface{})
	}
	playerInfo.GameID = gameID
	playerInfo.PlayerID = playerID
	playerInfo.CreatedAt = time.Now()
	playerInfo.UpdatedAt = time.Now()

	// 设置默认属性
	if playerInfo.Level == 0 {
		playerInfo.Level = 1
	}
	if playerInfo.Nickname == "" {
		playerInfo.Nickname = fmt.Sprintf("Player_%s", playerID[:8])
	}
	if playerInfo.Attributes["status"] == nil {
		playerInfo.Attributes["status"] = "active"
	}

	r.players[playerKey] = playerInfo

	// 保存到存储
	return r.saveToStorage()
}

// autoRegisterPlayer 自动注册访客玩家
func (r *InMemoryGameRegistry) autoRegisterPlayer(gameID, playerID string) error {
	playerInfo := &PlayerInfo{
		PlayerID: playerID,
		GameID:   gameID,
		Nickname: fmt.Sprintf("Guest_%s", playerID[:8]),
		Level:    1,
		Attributes: map[string]interface{}{
			"status":    "active",
			"isGuest":   true,
			"joinedAt":  time.Now(),
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	playerKey := fmt.Sprintf("%s:%s", gameID, playerID)
	r.players[playerKey] = playerInfo

	return r.saveToStorage()
}

// countPlayersInGame 计算游戏中的玩家数量
func (r *InMemoryGameRegistry) countPlayersInGame(gameID string) int {
	count := 0
	for key := range r.players {
		if len(key) > len(gameID) && key[:len(gameID)] == gameID {
			count++
		}
	}
	return count
}

// CreateGame 创建新游戏
func (r *InMemoryGameRegistry) CreateGame(gameInfo *GameInfo) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.games[gameInfo.GameID]; exists {
		return fmt.Errorf("game %s already exists", gameInfo.GameID)
	}

	gameInfo.CreatedAt = time.Now()
	gameInfo.UpdatedAt = time.Now()
	if gameInfo.Status == "" {
		gameInfo.Status = "active"
	}
	if gameInfo.MaxPlayers == 0 {
		gameInfo.MaxPlayers = 100
	}

	r.games[gameInfo.GameID] = gameInfo
	return r.saveToStorage()
}

// UpdatePlayerInfo 更新玩家信息
func (r *InMemoryGameRegistry) UpdatePlayerInfo(gameID, playerID string, updates map[string]interface{}) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	playerKey := fmt.Sprintf("%s:%s", gameID, playerID)
	player, exists := r.players[playerKey]
	if !exists {
		return fmt.Errorf("player %s not found in game %s", playerID, gameID)
	}

	// 更新允许的字段
	if nickname, ok := updates["nickname"].(string); ok {
		player.Nickname = nickname
	}
	if level, ok := updates["level"].(int); ok {
		player.Level = level
	}

	// 更新属性
	for key, value := range updates {
		if key != "nickname" && key != "level" {
			player.Attributes[key] = value
		}
	}

	player.UpdatedAt = time.Now()
	return r.saveToStorage()
}

// GetGameInfo 获取游戏信息
func (r *InMemoryGameRegistry) GetGameInfo(gameID string) (*GameInfo, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	game, exists := r.games[gameID]
	if !exists {
		return nil, fmt.Errorf("game %s not found", gameID)
	}

	// 返回副本
	gameCopy := *game
	return &gameCopy, nil
}

// ListGames 列出所有游戏
func (r *InMemoryGameRegistry) ListGames() []*GameInfo {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	games := make([]*GameInfo, 0, len(r.games))
	for _, game := range r.games {
		gameCopy := *game
		games = append(games, &gameCopy)
	}
	return games
}

// ListPlayersInGame 列出游戏中的所有玩家
func (r *InMemoryGameRegistry) ListPlayersInGame(gameID string) []*PlayerInfo {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var players []*PlayerInfo
	prefix := gameID + ":"
	for key, player := range r.players {
		if len(key) > len(prefix) && key[:len(prefix)] == prefix {
			playerCopy := *player
			players = append(players, &playerCopy)
		}
	}
	return players
}

// saveToStorage 保存数据到存储
func (r *InMemoryGameRegistry) saveToStorage() error {
	// 保存游戏信息
	gamesData, err := json.Marshal(r.games)
	if err != nil {
		return fmt.Errorf("marshal games: %w", err)
	}
	if err := r.store.Put("games", gamesData); err != nil {
		return fmt.Errorf("save games: %w", err)
	}

	// 保存玩家信息
	playersData, err := json.Marshal(r.players)
	if err != nil {
		return fmt.Errorf("marshal players: %w", err)
	}
	if err := r.store.Put("players", playersData); err != nil {
		return fmt.Errorf("save players: %w", err)
	}

	return nil
}

// loadFromStorage 从存储加载数据
func (r *InMemoryGameRegistry) loadFromStorage() error {
	// 加载游戏信息
	gamesData, err := r.store.Get("games")
	if err != nil && err != storage.ErrDataNotFound {
		return fmt.Errorf("load games: %w", err)
	}
	if err == nil {
		if err := json.Unmarshal(gamesData, &r.games); err != nil {
			return fmt.Errorf("unmarshal games: %w", err)
		}
	}

	// 加载玩家信息
	playersData, err := r.store.Get("players")
	if err != nil && err != storage.ErrDataNotFound {
		return fmt.Errorf("load players: %w", err)
	}
	if err == nil {
		if err := json.Unmarshal(playersData, &r.players); err != nil {
			return fmt.Errorf("unmarshal players: %w", err)
		}
	}

	return nil
}