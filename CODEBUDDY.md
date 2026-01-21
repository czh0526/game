# CODEBUDDY.md This file provides guidance to CodeBuddy when working with code in this repository.

## Project Overview

This is an Aries-based multiplayer game system combining decentralized identity (DID) and verifiable credentials (VC) with real-time HTML5 Canvas gaming. The system implements a custom `did:player` method for player authentication and uses W3C Verifiable Credentials for game achievements.

## Common Commands

### Development
- `make dev` - Start development server with hot reload on :8080
- `make watch` - Start with Air hot-reload (requires .air.toml configuration)
- `go run ./server/cmd -addr=:8080 -static=./client` - Manual dev server start

### Building
- `make build` - Build server binary to `bin/game-server`
- `make run` - Build and run production server
- `make deps` - Install and tidy Go dependencies

### Testing
- `make test` - Run all tests with verbose output
- `make test-coverage` - Generate HTML coverage report (coverage.html)
- `go test -v ./server/internal/...` - Test specific package

### Code Quality
- `make fmt` - Format all Go code
- `make lint` - Run golangci-lint (requires installation)
- `make clean` - Remove build artifacts and coverage files

### Testing the Game
Access http://localhost:8080 after starting server. The test workflow:
1. Click "创建身份" to create a DID
2. Click "连接游戏" to establish WebSocket connection
3. Use WASD keys to move the player
4. Type in chat box and press Enter to send messages

## Architecture

### High-Level Structure

The system follows a layered architecture with clear separation between client, server, identity, and credential management:

```
┌──────────────────────────────────────────────┐
│          HTML5 Canvas Client                 │
│  ┌────────────┐  ┌────────────┐             │
│  │GameEngine  │  │PlayerWallet│             │
│  │(Canvas)    │  │(DID/VC)    │             │
│  └────────────┘  └────────────┘             │
│  ┌────────────┐  ┌────────────┐             │
│  │GameNetwork │  │GameUI      │             │
│  │(WebSocket) │  │(Controls)  │             │
│  └────────────┘  └────────────┘             │
└──────────┬───────────────────────────────────┘
           │ WebSocket + REST API
┌──────────▼───────────────────────────────────┐
│          Go Game Server                      │
│  ┌──────────────────────────────────────┐   │
│  │  SimpleServer (game/simple_server.go)│   │
│  │  • WebSocket Handler                 │   │
│  │  • Room Management                   │   │
│  │  • Player State Sync                 │   │
│  └──────────────────────────────────────┘   │
│  ┌──────────────┐  ┌──────────────┐         │
│  │SimpleService │  │SimpleService │         │
│  │(DID)         │  │(VC)          │         │
│  └──────────────┘  └──────────────┘         │
└──────────────────────────────────────────────┘
```

### Core Components

**Client Architecture** (`client/src/`)

1. **GameEngine** (`engine/gameEngine.js`) - Core game loop and rendering
   - Canvas-based rendering with camera system
   - Input handling (keyboard/mouse)
   - Player/object rendering with animations
   - Mini-map and debug HUD

2. **GameNetwork** (`network/websocket.js`) - WebSocket communication
   - Connection management with auto-reconnect
   - Message routing and handlers
   - Game state synchronization
   - Chat system integration

3. **PlayerWallet** (`wallet/wallet.js`) - Identity and credential management
   - DID creation via `/api/did/create`
   - Local storage persistence
   - Credential collection and verification
   - Digital signatures

4. **GameUI** (`ui/ui.js`) - User interface layer
   - Connection status management
   - Notification system
   - Modal dialogs for wallet/tasks
   - Event handling bridge

**Server Architecture** (`server/`)

1. **Main Server** (`cmd/main.go`) - Entry point
   - HTTP server initialization
   - Service dependency injection
   - Graceful shutdown handling
   - Route configuration

2. **SimpleServer** (`internal/game/simple_server.go`) - Game logic
   - WebSocket connection upgrade and handling
   - Room-based multiplayer management
   - Player authentication and tracking
   - Message broadcasting

3. **DID Service** (`internal/did/simple_service.go`) - Identity management
   - Custom `did:player:gameID:playerID` format
   - DID document generation and storage
   - Key pair generation (Ed25519)
   - Resolution API

4. **VC Service** (`internal/vc/simple_service.go`) - Credential management
   - W3C Verifiable Credential issuance
   - Credential verification
   - Achievement/level/skill credential types
   - Issuer signature management

### Data Flow

**Player Authentication Flow:**
1. Client calls `POST /api/did/create` with gameId and nickname
2. Server generates DID (`did:player:default:uuid`), keypair, and DID document
3. Client stores DID, privateKey, publicKey in localStorage
4. Client connects via WebSocket to `/ws/game`
5. Client sends `auth` message with DID
6. Server validates DID and creates player session

**Game Loop Flow:**
1. Client GameEngine runs at 60 FPS via `requestAnimationFrame`
2. User input (WASD) updates local player position
3. Position changes trigger `player_move` WebSocket message
4. Server broadcasts position update to all room players
5. Other clients update their player maps
6. Canvas re-renders all players at new positions

**Credential Issuance Flow:**
1. Server detects game event (task completion, level up)
2. Server calls VC service to issue credential
3. Credential includes player DID, achievement data, issuer signature
4. Server sends `credential` message via WebSocket
5. Client wallet adds credential to local collection
6. Client displays notification and updates UI

### Key Patterns

**Message Protocol** - All WebSocket messages follow this structure:
```json
{
  "type": "message_type",
  "data": {...},
  "playerId": "optional",
  "roomId": "optional",
  "timestamp": "ISO8601"
}
```

**Room Management** - Players are organized into rooms (default: "default" room)
- Each room tracks connected players and game state
- Messages broadcast to all players in same room
- Room creation is automatic on first join

**State Synchronization** - Game uses optimistic client prediction
- Client updates local state immediately
- Server validates and broadcasts authoritative state
- Clients reconcile on receiving server updates

### Module Dependencies

The codebase has clear dependency layers:
- `cmd/main.go` depends on all internal services
- `internal/game` depends on `internal/did` and `internal/vc`
- `internal/vc` depends on `internal/did`
- `pkg/` contains reusable data structures with no internal dependencies

Client modules are loosely coupled through event callbacks:
- GameEngine receives callbacks from GameUI
- GameNetwork receives engine instance to sync state
- PlayerWallet is independent, accessed via GameUI

### Critical Implementation Details

**DID Format:** `did:player:{gameId}:{playerId}`
- gameId: Game instance identifier (e.g., "default")
- playerId: UUID generated for each player
- Example: `did:player:default:550e8400-e29b-41d4-a716-446655440000`

**WebSocket Message Types:**
- `auth` - Player authentication
- `join_room`/`leave_room` - Room management
- `player_move` - Position updates
- `player_action` - Game actions (interact, use_item)
- `chat` - Chat messages
- `game_state` - Full state sync
- `credential` - Credential notifications

**Credential Types:**
- AchievementCredential - Game accomplishments
- LevelCredential - Player progression
- SkillCredential - Unlocked abilities
- ItemCredential - Owned game items

### JavaScript File Structure

All client files were recently fixed for syntax errors. The files use:
- ES6 classes (no module imports, browser-native)
- Template literals for HTML generation
- LocalStorage for persistence
- Global instances (window.game, window.network, window.wallet)

**Important:** When editing client JS files:
- Maintain proper string escaping in template literals
- Ensure no trailing quotes or special characters
- Files are loaded via `<script>` tags in index.html order
- Dependencies: gameEngine.js → websocket.js → wallet.js → ui.js

### Testing Strategy

The system supports multiple testing approaches:
- Unit tests for individual services (`internal/*`)
- Integration tests for API endpoints
- Manual browser testing via test workflow (see Common Commands)
- WebSocket testing with browser DevTools or wscat

No automated E2E tests currently exist. Manual testing focuses on:
1. DID creation and resolution
2. WebSocket connection stability
3. Multi-client synchronization
4. Credential issuance and verification

### Development Notes

**Aries Integration:** The project includes `aries-framework-go/` as a replaced dependency in go.mod. Currently, the system uses simplified implementations (`Simple*` types) rather than full Aries framework features.

**Hot Reload:** Use `make watch` with Air for automatic server restart on file changes. Configure via `.air.toml`.

**Static Files:** The `-static` flag in server points to client directory. Server serves all files from this directory at `/`.

**CORS:** WebSocket upgrader allows all origins in current implementation. Production deployments should restrict this.

**Error Handling:** Client displays user-friendly notifications. Check browser console for detailed error logs. Server logs to stdout.
