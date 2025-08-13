# Makefile for Minecraft Resource Pack Server

# 变量定义
BINARY_NAME=resourcepack-server
BINARY_WINDOWS=$(BINARY_NAME).exe
BINARY_LINUX=$(BINARY_NAME)
BINARY_DARWIN=$(BINARY_NAME)

# Go 编译参数
LDFLAGS=-ldflags="-s -w"
GOOS?=$(shell go env GOOS)
GOARCH?=$(shell go env GOARCH)

# 默认目标
.PHONY: all
all: build

# 构建当前平台
.PHONY: build
build:
	@echo "正在构建 $(BINARY_NAME)..."
	go build $(LDFLAGS) -o $(BINARY_NAME) .
	@echo "构建完成: $(BINARY_NAME)"

# 构建 Windows 版本
.PHONY: build-windows
build-windows:
	@echo "正在构建 Windows 版本..."
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_WINDOWS) .
	@echo "构建完成: $(BINARY_WINDOWS)"

# 构建 Linux 版本
.PHONY: build-linux
build-linux:
	@echo "正在构建 Linux 版本..."
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_LINUX) .
	@echo "构建完成: $(BINARY_LINUX)"

# 构建 macOS 版本
.PHONY: build-darwin
build-darwin:
	@echo "正在构建 macOS 版本..."
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_DARWIN) .
	@echo "构建完成: $(BINARY_DARWIN)"

# 构建所有平台
.PHONY: build-all
build-all: build-windows build-linux build-darwin
	@echo "所有平台构建完成"

# 清理构建文件
.PHONY: clean
clean:
	@echo "正在清理构建文件..."
	@rm -f $(BINARY_NAME) $(BINARY_WINDOWS) $(BINARY_LINUX) $(BINARY_DARWIN)
	@echo "清理完成"

# 下载依赖
.PHONY: deps
deps:
	@echo "正在下载依赖..."
	go mod tidy
	go mod download
	@echo "依赖下载完成"

# 运行测试
.PHONY: test
test:
	@echo "正在运行测试..."
	go test -v ./...
	@echo "测试完成"

# 代码格式化
.PHONY: fmt
fmt:
	@echo "正在格式化代码..."
	go fmt ./...
	@echo "代码格式化完成"

# 代码检查
.PHONY: lint
lint:
	@echo "正在检查代码..."
	golangci-lint run
	@echo "代码检查完成"

# 运行程序
.PHONY: run
run: build
	@echo "正在启动服务器..."
	./$(BINARY_NAME)

# 帮助信息
.PHONY: help
help:
	@echo "可用的构建目标:"
	@echo "  build         - 构建当前平台版本"
	@echo "  build-windows - 构建 Windows 版本"
	@echo "  build-linux   - 构建 Linux 版本"
	@echo "  build-darwin  - 构建 macOS 版本"
	@echo "  build-all     - 构建所有平台版本"
	@echo "  clean         - 清理构建文件"
	@echo "  deps          - 下载依赖"
	@echo "  test          - 运行测试"
	@echo "  fmt           - 格式化代码"
	@echo "  lint          - 检查代码"
	@echo "  run           - 构建并运行"
	@echo "  help          - 显示此帮助信息"
