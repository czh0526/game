# Aries Game System Makefile

.PHONY: build run test clean deps dev

# 默认目标
all: build

# 安装依赖
deps:
	go mod tidy
	go mod download

# 构建服务器
build:
	go build -o bin/game-server ./server/cmd

# 运行开发服务器
dev:
	go run ./server/cmd -addr=:8080 -static=./client

# 运行生产服务器
run: build
	./bin/game-server -addr=:8080 -static=./client

# 运行测试
test:
	go test -v ./...

# 运行测试并生成覆盖率报告
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# 清理构建文件
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# 格式化代码
fmt:
	go fmt ./...

# 代码检查
lint:
	golangci-lint run

# 启动开发环境（包含热重载）
watch:
	air -c .air.toml

# Docker构建
docker-build:
	docker build -t aries-game-system .

# Docker运行
docker-run:
	docker run -p 8080:8080 aries-game-system

# 生成API文档
docs:
	swag init -g server/cmd/main.go -o docs/

# 数据库迁移（如果使用数据库）
migrate-up:
	migrate -path migrations -database "sqlite3://game.db" up

migrate-down:
	migrate -path migrations -database "sqlite3://game.db" down

# 帮助信息
help:
	@echo "Available targets:"
	@echo "  deps         - Install dependencies"
	@echo "  build        - Build the server"
	@echo "  dev          - Run development server"
	@echo "  run          - Run production server"
	@echo "  test         - Run tests"
	@echo "  test-coverage- Run tests with coverage"
	@echo "  clean        - Clean build files"
	@echo "  fmt          - Format code"
	@echo "  lint         - Run linter"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run Docker container"