# 编译问题修复指南

## 问题原因

错误信息：
```
Binary was compiled with 'CGO_ENABLED=0', go-sqlite3 requires cgo to work.
```

**原因**：x-ui 和 backend-proxy 都使用了 `gorm.io/driver/sqlite`，它依赖 `go-sqlite3`，这个库**必须启用 CGO** 才能工作。

## 解决方案

### 方案 1：在服务器上直接编译（最简单，推荐）

在目标服务器上编译，这样不需要处理交叉编译的复杂性：

```bash
# SSH 登录服务器
ssh root@your-server

# 安装 Go（如果未安装）
# Ubuntu/Debian:
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# 克隆或上传代码到服务器
cd /tmp
git clone [your-repo]  # 或上传代码

# 编译 x-ui
cd x-ui
go build -ldflags="-s -w" -o x-ui ./

# 编译 backend-proxy
cd backend-proxy
go build -ldflags="-s -w" -o backend-proxy ./

# 移动到部署目录
sudo mv x-ui /usr/local/x-ui/
sudo mv backend-proxy /usr/local/x-ui/backend-proxy/
```

### 方案 2：本地交叉编译（需要安装工具）

#### macOS 安装交叉编译工具

```bash
# 安装 musl-cross（推荐）
brew install filosottile/musl-cross/musl-cross

# 或者使用 musl-toolchain
brew install musl-cross

# 然后设置环境变量
export CC=x86_64-linux-musl-gcc
export CGO_ENABLED=1
export GOOS=linux
export GOARCH=amd64
go build -ldflags="-s -w" -o x-ui ./
```

#### Linux 交叉编译

```bash
# Ubuntu/Debian 安装交叉编译器
sudo apt-get install gcc-x86-64-linux-gnu

export CC=x86_64-linux-gnu-gcc
export CGO_ENABLED=1
export GOOS=linux
export GOARCH=amd64
go build -ldflags="-s -w" -o x-ui ./
```

### 方案 3：使用 Docker 编译（推荐用于复杂场景）

创建 `Dockerfile.build`：

```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /build

# 安装依赖
RUN apk add --no-cache gcc musl-dev sqlite-dev

# 复制代码
COPY . .

# 编译
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o x-ui ./
RUN cd backend-proxy && CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o backend-proxy ./
```

编译：
```bash
docker build -f Dockerfile.build -t x-ui-build .
docker create --name temp x-ui-build
docker cp temp:/build/x-ui ./x-ui
docker cp temp:/build/backend-proxy ./backend-proxy
docker rm temp
```

### 方案 4：修改部署脚本（已修复）

部署脚本已经修复，会尝试启用 CGO。但如果你的 macOS 没有安装交叉编译工具，可能还是会失败。

**快速修复**：修改 `.deploy.conf`，添加一个选项：

```bash
# 如果本地编译失败，是否在服务器上编译
COMPILE_ON_SERVER="no"  # yes 或 no
```

## 当前修复

我已经修复了 `deploy.sh`，现在会：
1. 启用 CGO（`CGO_ENABLED=1`）
2. 尝试检测并使用交叉编译工具
3. 如果失败，会给出提示

## 立即解决当前问题

### 方法 1：在服务器上编译（最快）

```bash
# 在服务器上
cd /tmp
git clone [your-repo] x-ui-src  # 或上传代码
cd x-ui-src

# 安装 Go（如果没有）
# 然后编译
go build -ldflags="-s -w" -o x-ui ./
cd backend-proxy
go build -ldflags="-s -w" -o backend-proxy ./

# 替换文件
sudo systemctl stop x-ui
sudo systemctl stop backend-proxy
sudo mv x-ui /usr/local/x-ui/
sudo mv backend-proxy /usr/local/x-ui/backend-proxy/
sudo systemctl start x-ui
sudo systemctl start backend-proxy
```

### 方法 2：在本地重新编译并上传

如果 macOS 有交叉编译工具，重新运行：
```bash
./deploy.sh
```

## 验证修复

编译成功后，在服务器上：
```bash
# 检查文件
file /usr/local/x-ui/x-ui

# 应该显示类似：
# /usr/local/x-ui/x-ui: ELF 64-bit LSB executable, x86-64, version 1 (SYSV), dynamically linked, ...

# 启动服务
sudo systemctl restart x-ui
sudo systemctl status x-ui

# 查看日志
journalctl -u x-ui -f
```

## 长期解决方案

建议：
1. **在 CI/CD 中编译**：使用 GitHub Actions 或其他 CI 工具在 Linux 环境中编译
2. **使用 Docker**：在 Docker 容器中编译，确保环境一致
3. **服务器编译**：在服务器上直接编译（最简单）

