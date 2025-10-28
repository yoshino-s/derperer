# Derperer

![GitHub Release](https://img.shields.io/github/v/release/yoshino-s/derperer) [![goreleaser](https://github.com/yoshino-s/derperer/actions/workflows/publish.yaml/badge.svg)](https://github.com/yoshino-s/derperer/actions/workflows/publish.yaml) ![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/yoshino-s/derperer)

**语言版本:** [English](README.MD) | [中文](README_zh.md)

Derperer 是 [derper](https://pkg.go.dev/tailscale.com/cmd/derper) 的母项目 - 一个用于管理和测试 DERP（指定加密数据包中继）服务器的工具，提供发现、速度测试和监控功能。

## 功能特性

- 🔍 **DERP服务器发现**: 使用FOFA搜索引擎自动发现和监控DERP服务器
- 🚀 **速度测试**: 测试到DERP服务器的连接速度和延迟
- 📊 **HTTP API**: 带有Swagger文档的RESTful API，支持程序化访问
- 📈 **实时监控**: 持续健康检查和监控DERP端点
- 🇨🇳 **中国区域支持**: 对中国区域DERP服务器的特殊支持
- 📝 **灵活配置**: 支持YAML、JSON、TOML和环境变量配置

## 安装

### 从源码编译

```bash
git clone https://github.com/yoshino-s/derperer.git
cd derperer
go build -o derperer .
```

### 使用 Go Install

```bash
go install github.com/yoshino-s/derperer@latest
```

### 使用 Docker

```bash
# 拉取并运行最新镜像
docker run --rm -p 8080:8080 ghcr.io/yoshino-s/derperer:latest serve

# 运行速度测试
docker run --rm ghcr.io/yoshino-s/derperer:latest speedtest
```

### 使用 Docker Compose

```bash
# 克隆仓库并使用docker-compose
git clone https://github.com/yoshino-s/derperer.git
cd derperer
docker-compose up -d
```

## 使用方法

### 快速开始

```bash
# 运行服务器
derperer serve

# 运行速度测试
derperer speedtest --derp_region_id 1

# 生成配置文件
derperer --generate-config.enable --generate-config.path config.yaml
```

### 命令

#### 主命令

```bash
derperer [command]
```

**可用命令:**
- `completion` - 为指定shell生成自动补全脚本
- `help` - 显示任何命令的帮助信息
- `serve` - 启动HTTP服务器
- `speedtest` - 运行速度测试
- `version` - 显示版本信息

#### Serve 命令

启动HTTP服务器进行DERP服务器发现和监控：

```bash
derperer serve [flags]
```

**参数:**
- `--derperer.check_concurrency int` - 并发测试数量 (默认 10)
- `--derperer.check_duration duration` - 检查节点的持续时间 (默认 10s)
- `--derperer.cn` - 仅获取中国区域节点
- `--derperer.fetch_limit int` - FOFA结果获取限制 (默认 100)
- `--derperer.recheck_interval duration` - 重新检查废弃节点的间隔 (默认 10s)
- `--derperer.refetch_interval duration` - 重新获取数据的间隔 (默认 10m0s)
- `--fofa.email string` - FOFA邮箱
- `--fofa.endpoint string` - FOFA端点 (默认 "https://fofa.info/api/v1")
- `--fofa.key string` - FOFA密钥
- `--http.addr string` - HTTP监听地址 (默认 ":8080")
- `--http.behind_proxy` - HTTP代理后方
- `--http.external_url string` - 外部URL (默认 "http://127.0.0.1:8080")
- `--http.feature uint16` - HTTP功能 (默认 30)
- `--http.log` - 启用HTTP日志
- `--http.otel` - 启用OpenTelemetry
- `--http.response_trace_id` - 在响应头中启用x-trace-id

#### 速度测试命令

对DERP服务器运行速度测试：

```bash
derperer speedtest [flags]
```

**参数:**
- `--derp_map_url string` - DERP映射URL (默认 "https://controlplane.tailscale.com/derpmap/default")
- `--derp_region_id int` - DERP区域ID
- `--duration duration` - 测试持续时间 (默认 30s)

### 全局参数

- `--generate-config.enable` - 启用配置生成
- `--generate-config.format string` - 生成配置格式，可选 json, yaml, toml, env (默认 "yaml")
- `--generate-config.path string` - 生成配置路径
- `--log.file string` - 日志文件路径
- `--log.format string` - 日志格式，可选 json, console，空为默认 (开发环境用console，生产环境用json)
- `--log.level string` - 日志级别 (默认 "info")
- `--log.levels.console string` - 控制台日志级别，空表示与log.level相同
- `--log.levels.file string` - 文件日志级别，空表示与log.level相同
- `--log.rotate.enable` - 启用日志轮转
- `--log.rotate.max_age int` - 日志文件最大天数 (默认 28)
- `--log.rotate.max_backups int` - 日志文件最大备份数 (默认 3)
- `--log.rotate.max_size int` - 日志文件最大大小（MB） (默认 500)

## 配置

Derperer支持通过YAML、JSON、TOML文件或环境变量进行配置。应用程序会自动在以下位置搜索配置文件（按优先级顺序）：

1. 当前工作目录：`./derperer.yaml`, `./derperer.json`, `./derperer.toml`
2. 用户配置目录：`~/.config/derperer/derperer.yaml`
3. 系统配置目录：`/etc/derperer/derperer.yaml`
4. 使用 `DERPERER_` 前缀的环境变量

您也可以使用 `--config` 标志指定自定义配置文件或生成示例配置文件：

```bash
# 生成示例配置
derperer --generate-config.enable --generate-config.path derperer.yaml

# 使用自定义配置文件
derperer serve --config /path/to/your/config.yaml
```

配置示例 (`derperer.yaml`)：

```yaml
derperer:
  check_duration: 10s
  recheck_interval: 10s
  refetch_interval: 10m0s
  check_concurrency: 10
  cn: false  # 设置为true仅限中国区域
  fetch_limit: 100

fofa:
  email: "your-email@example.com"
  key: "your-fofa-key"
  endpoint: "https://fofa.info/api/v1"

http:
  addr: ":8080"
  external_url: "http://127.0.0.1:8080"
  feature: 30
  log: false
  otel: false
  response_trace_id: false

log:
  level: "info"
  format: ""  # 空表示默认
  file: ""    # 空表示输出到标准输出
  rotate:
    enable: false
    max_age: 28
    max_backups: 3
    max_size: 500
```

## API文档

运行服务器时，Swagger文档可在以下地址访问：
- Swagger UI: `http://localhost:8080/docs/`
- Swagger JSON: `http://localhost:8080/docs/swagger.json`
- Swagger YAML: `http://localhost:8080/docs/swagger.yaml`

## 使用示例

### 基本服务器启动

```bash
# 使用默认设置启动服务器
derperer serve

# 使用自定义端口启动服务器
derperer serve --http.addr :9090

# 仅启动中国区域服务器
derperer serve --derperer.cn
```

### 速度测试

```bash
# 测试默认区域
derperer speedtest

# 测试特定区域
derperer speedtest --derp_region_id 5

# 测试60秒
derperer speedtest --duration 60s

# 使用自定义DERP映射测试
derperer speedtest --derp_map_url https://example.com/derpmap
```

### 配置管理

```bash
# 生成YAML配置
derperer --generate-config.enable --generate-config.format yaml --generate-config.path config.yaml

# 生成JSON配置
derperer --generate-config.enable --generate-config.format json --generate-config.path config.json

# 使用配置文件运行
derperer serve --config config.yaml
```

### Docker Compose 使用

附带的 `docker-compose.yml` 提供了运行Derperer的完整设置：

```bash
# 启动服务
docker-compose up -d

# 查看日志
docker-compose logs -f

# 停止服务
docker-compose down

# 更新到最新版本
docker-compose pull && docker-compose up -d
```

**配置FOFA凭据:**

1. 编辑 `docker-compose.yml` 文件并取消注释FOFA环境变量：
   ```yaml
   environment:
     - DERPERER_FOFA_EMAIL=your-email@example.com
     - DERPERER_FOFA_KEY=your-fofa-key
   ```

2. 或在同一目录下创建 `.env` 文件：
   ```env
   DERPERER_FOFA_EMAIL=your-email@example.com
   DERPERER_FOFA_KEY=your-fofa-key
   DERPERER_DERPERER_CN=false
   ```

**使用配置文件的替代配置:**

在 `docker-compose.yml` 中取消注释卷挂载并创建本地配置文件：

```yaml
volumes:
  - ./derperer.yaml:/app/derperer.yaml:ro
```

## 系统要求

- Go 1.25.2 或更高版本
- FOFA账户（用于服务器发现功能）

## 贡献

欢迎贡献！请随时提交Pull Request。

## 许可证

本项目根据仓库中指定的条款获得许可。

## 相关项目

- [Tailscale DERP](https://pkg.go.dev/tailscale.com/cmd/derper) - 原始DERP服务器实现