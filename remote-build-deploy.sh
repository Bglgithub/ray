#!/bin/bash

# 服务器端自动编译和部署脚本
# 使用方法: ./remote-build-deploy.sh

set -e  # 遇到错误立即退出

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

# 配置
GIT_REPO_URL="https://github.com/Bglgithub/ray.git"  # Git 仓库地址，如果为空则需要手动设置
BRANCH="${BRANCH:-main}"           # Git 分支，默认 main
BUILD_DIR="/tmp/x-ui-build"         # 临时构建目录
XUI_DEPLOY_PATH="/usr/local/x-ui"
BACKEND_PROXY_DEPLOY_PATH="/usr/local/x-ui/backend-proxy"
XUI_SERVICE="x-ui"
BACKEND_PROXY_SERVICE="backend-proxy"

echo "=========================================="
print_info "开始自动编译和部署"
echo "=========================================="
echo ""

# 检查并安装 C 编译器（gcc）
print_info "检查 C 编译器（gcc）..."
if ! command -v gcc &> /dev/null; then
    print_warning "gcc 未安装，正在安装..."
    
    # 检测 Linux 发行版
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        OS_ID="$ID"
    else
        OS_ID="unknown"
    fi
    
    # 根据发行版安装 gcc
    case "$OS_ID" in
        ubuntu|debian)
            print_info "检测到 Ubuntu/Debian，使用 apt 安装..."
            sudo apt-get update -qq
            sudo apt-get install -y -qq gcc libc6-dev build-essential
            ;;
        centos|rhel|fedora)
            print_info "检测到 CentOS/RHEL/Fedora，使用 yum/dnf 安装..."
            if command -v dnf &> /dev/null; then
                sudo dnf install -y -q gcc glibc-devel
            else
                sudo yum install -y -q gcc glibc-devel
            fi
            ;;
        *)
            print_error "无法自动识别 Linux 发行版，请手动安装 gcc"
            print_info "Ubuntu/Debian: sudo apt-get install -y gcc build-essential"
            print_info "CentOS/RHEL: sudo yum install -y gcc glibc-devel"
            exit 1
            ;;
    esac
    
    # 验证安装
    if ! command -v gcc &> /dev/null; then
        print_error "gcc 安装失败，请手动安装"
        exit 1
    fi
    print_success "gcc 安装成功: $(gcc --version | head -1)"
else
    print_success "gcc 已安装: $(gcc --version | head -1)"
fi

# 检查 Go 环境
print_info "检查 Go 环境..."
if ! command -v go &> /dev/null; then
    print_error "Go 未安装"
    print_info "正在安装 Go..."
    
    # 安装 Go
    cd /tmp
    GO_VERSION="1.21.6"
    GO_ARCH="amd64"
    wget -q https://go.dev/dl/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz
    sudo tar -C /usr/local -xzf go${GO_VERSION}.linux-${GO_ARCH}.tar.gz
    export PATH=$PATH:/usr/local/go/bin
    rm -f go${GO_VERSION}.linux-${GO_ARCH}.tar.gz
    
    # 验证安装
    if ! command -v go &> /dev/null; then
        print_error "Go 安装失败，请手动安装"
        exit 1
    fi
    print_success "Go 安装成功: $(go version)"
else
    print_success "Go 已安装: $(go version)"
fi

# 设置 Go 环境
export PATH=$PATH:/usr/local/go/bin
export CGO_ENABLED=1
print_info "启用 CGO 支持（CGO_ENABLED=1）"

# 获取 Git 仓库地址（如果未设置）
if [ -z "$GIT_REPO_URL" ]; then
    print_warning "GIT_REPO_URL 环境变量未设置"
    read -p "请输入 Git 仓库地址: " GIT_REPO_URL
    if [ -z "$GIT_REPO_URL" ]; then
        print_error "Git 仓库地址不能为空"
        exit 1
    fi
fi

# 清理旧的构建目录
print_info "清理旧的构建目录..."
rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR"

# 克隆或更新代码
print_info "获取最新代码..."
cd "$BUILD_DIR"

if [ -d ".git" ]; then
    print_info "更新代码库..."
    git fetch origin
    git reset --hard origin/$BRANCH
    git clean -fd
else
    print_info "克隆代码库..."
    git clone -b "$BRANCH" "$GIT_REPO_URL" .
fi

print_success "代码获取完成"

# 编译 x-ui
print_info "编译 x-ui 服务..."
cd "$BUILD_DIR"
go mod download
go build -ldflags="-s -w" -o x-ui ./

if [ $? -eq 0 ]; then
    print_success "x-ui 编译成功"
else
    print_error "x-ui 编译失败"
    exit 1
fi

# 编译 backend-proxy
print_info "编译 backend-proxy 服务..."
cd "$BUILD_DIR/backend-proxy"
go mod download
go build -ldflags="-s -w" -o backend-proxy ./

if [ $? -eq 0 ]; then
    print_success "backend-proxy 编译成功"
else
    print_error "backend-proxy 编译失败"
    exit 1
fi

# 停止服务
print_info "停止服务..."
sudo systemctl stop $XUI_SERVICE 2>/dev/null || true
sudo systemctl stop $BACKEND_PROXY_SERVICE 2>/dev/null || true
sleep 2

# 备份旧文件
print_info "备份旧文件..."
BACKUP_DIR="/tmp/x-ui-backup-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$BACKUP_DIR"

if [ -f "$XUI_DEPLOY_PATH/x-ui" ]; then
    sudo cp "$XUI_DEPLOY_PATH/x-ui" "$BACKUP_DIR/x-ui" 2>/dev/null || true
    print_info "已备份 x-ui"
fi

if [ -f "$BACKEND_PROXY_DEPLOY_PATH/backend-proxy" ]; then
    sudo cp "$BACKEND_PROXY_DEPLOY_PATH/backend-proxy" "$BACKUP_DIR/backend-proxy" 2>/dev/null || true
    print_info "已备份 backend-proxy"
fi

# 部署新文件
print_info "部署新文件..."

# 部署 x-ui
sudo mkdir -p "$XUI_DEPLOY_PATH"
sudo cp "$BUILD_DIR/x-ui" "$XUI_DEPLOY_PATH/x-ui"
sudo chmod +x "$XUI_DEPLOY_PATH/x-ui"
sudo chown root:root "$XUI_DEPLOY_PATH/x-ui"

# 复制 bin 目录（包含 config.json、geosite.dat、geoip.dat、xray 二进制等）
if [ -d "$BUILD_DIR/bin" ]; then
    print_info "复制 bin 目录..."
    sudo mkdir -p "$XUI_DEPLOY_PATH/bin"
    
    # 复制 bin 目录中的所有文件和子目录
    sudo cp -r "$BUILD_DIR/bin"/* "$XUI_DEPLOY_PATH/bin/" 2>/dev/null || true
    
    # 设置权限
    sudo chown -R root:root "$XUI_DEPLOY_PATH/bin" 2>/dev/null || true
    sudo chmod -R 755 "$XUI_DEPLOY_PATH/bin" 2>/dev/null || true
    
    # 确保 xray 二进制有执行权限
    if [ -f "$XUI_DEPLOY_PATH/bin/xray-linux-amd64" ]; then
        sudo chmod +x "$XUI_DEPLOY_PATH/bin/xray-linux-amd64"
    fi
    if [ -f "$XUI_DEPLOY_PATH/bin/xray-linux-arm64" ]; then
        sudo chmod +x "$XUI_DEPLOY_PATH/bin/xray-linux-arm64"
    fi
    
    print_success "bin 目录复制完成"
else
    print_warning "未找到 bin 目录，请确保 bin 目录已存在于代码仓库中"
fi

print_success "x-ui 部署完成"

# 部署 backend-proxy
sudo mkdir -p "$BACKEND_PROXY_DEPLOY_PATH"
sudo cp "$BUILD_DIR/backend-proxy/backend-proxy" "$BACKEND_PROXY_DEPLOY_PATH/backend-proxy"
sudo chmod +x "$BACKEND_PROXY_DEPLOY_PATH/backend-proxy"
sudo chown root:root "$BACKEND_PROXY_DEPLOY_PATH/backend-proxy"

# 备份并更新配置文件示例（如果需要）
if [ -f "$BUILD_DIR/backend-proxy/env.example" ]; then
    if [ ! -f "$BACKEND_PROXY_DEPLOY_PATH/.env" ]; then
        print_info "创建 backend-proxy 配置文件..."
        sudo cp "$BUILD_DIR/backend-proxy/env.example" "$BACKEND_PROXY_DEPLOY_PATH/.env"
        sudo chown root:root "$BACKEND_PROXY_DEPLOY_PATH/.env"
        print_warning "请编辑配置文件: $BACKEND_PROXY_DEPLOY_PATH/.env"
    fi
fi

print_success "backend-proxy 部署完成"

# 创建或更新 systemd 服务
print_info "创建 systemd 服务..."

# 创建 x-ui systemd 服务
XUI_SERVICE_FILE="/etc/systemd/system/${XUI_SERVICE}.service"
sudo tee "$XUI_SERVICE_FILE" > /dev/null << EOF
[Unit]
Description=x-ui Service
After=network.target

[Service]
Type=simple
WorkingDirectory=${XUI_DEPLOY_PATH}
ExecStart=${XUI_DEPLOY_PATH}/x-ui
Restart=always
RestartSec=5s
User=root

[Install]
WantedBy=multi-user.target
EOF

# 创建 backend-proxy systemd 服务
BACKEND_PROXY_SERVICE_FILE="/etc/systemd/system/${BACKEND_PROXY_SERVICE}.service"
sudo tee "$BACKEND_PROXY_SERVICE_FILE" > /dev/null << EOF
[Unit]
Description=Backend Proxy Service
After=network.target
Requires=network.target

[Service]
Type=simple
WorkingDirectory=${BACKEND_PROXY_DEPLOY_PATH}
ExecStart=${BACKEND_PROXY_DEPLOY_PATH}/backend-proxy
Restart=always
RestartSec=5s
Environment="XUI_DB_PATH=/etc/x-ui/x-ui.db"
Environment="XUI_SERVER_URL=http://localhost:54321"
Environment="PROXY_PORT=8080"

[Install]
WantedBy=multi-user.target
EOF

sudo chmod 644 "$XUI_SERVICE_FILE" "$BACKEND_PROXY_SERVICE_FILE"
print_success "systemd 服务文件创建完成"

# 启动服务
print_info "启动服务..."

sudo systemctl daemon-reload

# 启动 x-ui
if sudo systemctl start $XUI_SERVICE; then
    sleep 2
    if sudo systemctl is-active --quiet $XUI_SERVICE; then
        print_success "x-ui 服务启动成功"
    else
        print_error "x-ui 服务启动失败"
        print_info "查看日志: journalctl -u $XUI_SERVICE -n 50"
    fi
else
    print_error "x-ui 服务启动失败"
fi

# 启动 backend-proxy
if sudo systemctl start $BACKEND_PROXY_SERVICE; then
    sleep 2
    if sudo systemctl is-active --quiet $BACKEND_PROXY_SERVICE; then
        print_success "backend-proxy 服务启动成功"
    else
        print_error "backend-proxy 服务启动失败"
        print_info "查看日志: journalctl -u $BACKEND_PROXY_SERVICE -n 50"
    fi
else
    print_error "backend-proxy 服务启动失败"
fi

# 显示服务状态
echo ""
print_info "服务状态:"
sudo systemctl status $XUI_SERVICE --no-pager -l | head -10
echo ""
sudo systemctl status $BACKEND_PROXY_SERVICE --no-pager -l | head -10

# 清理临时文件（可选）
read -p "是否清理临时构建目录？(y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    rm -rf "$BUILD_DIR"
    print_success "临时文件已清理"
else
    print_info "保留构建目录: $BUILD_DIR"
fi

echo ""
echo "=========================================="
print_success "部署完成！"
echo "=========================================="
echo ""
print_info "备份文件位置: $BACKUP_DIR"
print_info "如果出现问题，可以从备份恢复:"
echo "  sudo cp $BACKUP_DIR/x-ui $XUI_DEPLOY_PATH/x-ui"
echo "  sudo cp $BACKUP_DIR/backend-proxy $BACKEND_PROXY_DEPLOY_PATH/backend-proxy"
echo ""
print_info "查看服务日志:"
echo "  journalctl -u $XUI_SERVICE -f"
echo "  journalctl -u $BACKEND_PROXY_SERVICE -f"
echo ""

