module github.com/czh0526/game

go 1.21.0

toolchain go1.22.4

require (
	github.com/go-sql-driver/mysql v1.9.3
	github.com/google/uuid v1.4.0
	github.com/gorilla/websocket v1.5.3
)

require filippo.io/edwards25519 v1.1.0 // indirect

replace github.com/hyperledger/aries-framework-go => ./aries-framework-go
