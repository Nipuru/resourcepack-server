<div align="center">
    <h1>resourcepack-server</h1>
    <h5>高性能 Minecraft 资源包分发服务器，支持自动文件监控和实时更新。</h5>
</div>

## ✨ 特性

- 基于 Go 语言和 Gin 框架的高性能 Web 服务器
- 支持 ZIP 文件和目录形式的资源包
- 实时监控资源包目录变化，自动更新
- 提供完整的资源包管理 API
- 内置调试界面，实时查看服务器状态

## 🚀 快速开始

### 1. 环境要求

- Go 1.21 或更高版本

### 2. 启动服务器

```bash
# 下载依赖
go mod tidy

# 编译程序
go build -o resourcepack-server .

# 运行程序
./resourcepack-server
```

## 🔍 文件监控

### 自动检测变化

服务器会自动监控资源包目录，检测以下变化：

- 添加新的 ZIP 文件或目录
- 移除资源包文件或目录  
- 更新 ZIP 文件或 pack.mcmeta
- 重命名或移动资源包

### 手动重新扫描

如果需要手动触发重新扫描，可以调用 API：

```bash
curl -X POST http://localhost:8080/api/rescan
```

## 🌐 API 接口

### 资源包列表
```
GET /api/packs
```

### 获取特定资源包
```
GET /api/packs/{name}
```

### 下载资源包
```
GET /download/{name}
```

### 获取资源包 Hash
```
GET /hash/{name}
```

### 手动重新扫描
```
POST /api/rescan
```

### 调试信息
```
GET /debug
```

## 📁 资源包格式

### ZIP 文件
- 直接上传 `.zip` 文件到资源包目录
- 自动检测并解析 `pack.mcmeta`

### 目录形式
- 创建包含 `pack.mcmeta` 的目录
- 服务器会动态压缩并提供下载

## 📝 配置说明

程序启动时会自动创建配置文件 `config.toml`，用户可以根据需要修改配置项。
## 📝 注意事项

1. 确保服务器有读取资源包目录的权限
2. 大量文件变化时会有短暂延迟
3. 扫描冷却时间防止频繁扫描，可根据需要调整

### 端口被占用
修改 `config.toml` 中的端口设置

## 📦 构建

### 使用 Makefile (推荐)
```bash
# 构建当前平台
make build

# 构建特定平台
make build-windows    # Windows
make build-linux      # Linux
make build-darwin     # macOS

# 构建所有平台
make build-all

# 查看所有可用命令
make help
```

### 手动构建
```bash
go build -ldflags="-s -w" -o resourcepack-server .
```
