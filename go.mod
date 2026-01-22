module github.com/czh0526/game

go 1.22

toolchain go1.22.4

require (
	github.com/google/uuid v1.4.0
	github.com/gorilla/websocket v1.5.3
	github.com/hyperledger/aries-framework-go v0.0.0-00010101000000-000000000000
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/go-sql-driver/mysql v1.9.3 // indirect
	github.com/hyperledger/aries-framework-go-ext/component/storage/mysql v0.0.0-20240327164505-1a43f7c44255 // indirect
	github.com/hyperledger/aries-framework-go/spi v0.0.0-20230517133327-301aa0597250 // indirect
	github.com/hyperledger/aries-framework-go/test/component v0.0.0-20220330140627-07042d78580c // indirect
	github.com/ory/dockertest/v3 v3.12.0 // indirect
	github.com/stretchr/testify v1.10.0 // indirect
)

replace github.com/hyperledger/aries-framework-go => ./aries-framework-go
