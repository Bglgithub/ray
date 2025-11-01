#!/bin/bash

# x-ui 和 backend-proxy 一键部署脚本
# 使用方法: ./deploy.sh

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

# 检查配置文件
CONFIG_FILE=".deploy.conf"
if [ ! -f "$CONFIG_FILE" ]; then
    print_error "配置文件不存在: $CONFIG_FILE"
    print_info "正在创建示例配置文件..."
    
    cat > "$CONFIG_FILE" << 'EOF'
# 部署配置文件
# 远程服务器信息
REMOTE_HOST="16.176.193.236"
REMOTE_USER="root"
REMOTE_PORT="22"

# 部署路径
XUI_DEPLOY_PATH="/usr/local/x-ui"
BACKEND_PROXY_DEPLOY_PATH="/usr/local/x-ui/backend-proxy"

# x-ui 配置
XUI_SERVICE_NAME="x-ui"
XUI_BINARY_NAME="x-ui"

# backend-proxy 配置
BACKEND_PROXY_SERVICE_NAME="backend-proxy"
BACKEND_PROXY_BINARY_NAME="backend-proxy"
BACKEND_PROXY_PORT="8080"

# 是否创建 systemd 服务（需要 root 权限）
CREATE_SYSTEMD_SERVICE="yes"

# 编译选项
BUILD_ARCH="amd64"  # amd64, arm64, 386
BUILD_OS="linux"    # linux, darwin, windows

# SSH 选项
SSH_KEY="/Users/guolinbao/Desktop/vps/ula-rsa.pem"  # 如果使用密钥，填写密钥路径，如: ~/.ssh/id_rsa
EOF
    
    print_success "配置文件已创建: $CONFIG_FILE"
    print_info "请编辑配置文件后再运行脚本"
    exit 1
fi

# 加载配置
source "$CONFIG_FILE"

# 验证必要配置
if [ "$REMOTE_HOST" = "your-server-ip-or-domain" ] || [ -z "$REMOTE_HOST" ]; then
    print_error "请先配置 REMOTE_HOST"
    exit 1
fi

# 构建 SSH 命令
SSH_CMD="ssh"
if [ -n "$SSH_KEY" ]; then
    SSH_CMD="$SSH_CMD -i $SSH_KEY"
fi
if [ -n "$REMOTE_PORT" ]; then
    SSH_CMD="$SSH_CMD -p $REMOTE_PORT"
fi
SSH_CMD="$SSH_CMD ${REMOTE_USER}@${REMOTE_HOST}"

SCP_CMD="scp"
if [ -n "$SSH_KEY" ]; then
    SCP_CMD="$SCP_CMD -i $SSH_KEY"
fi
if [ -n "$REMOTE_PORT" ]; then
    SCP_CMD="$SCP_CMD -P $REMOTE_PORT"
fi

echo "=========================================="
print_info "开始部署流程"
echo "=========================================="
echo "目标服务器: ${REMOTE_USER}@${REMOTE_HOST}:${REMOTE_PORT:-22}"
echo "部署路径:"
echo "  - x-ui: $XUI_DEPLOY_PATH"
echo "  - backend-proxy: $BACKEND_PROXY_DEPLOY_PATH"
echo ""

# 1. 编译 x-ui
print_info "步骤 1/6: 编译 x-ui 服务..."
cd "$(dirname "$0")"
if [ ! -d ".git" ]; then
    print_error "不在项目根目录"
    exit 1
fi

export GOOS="$BUILD_OS"
export GOARCH="$BUILD_ARCH"

# 注意：x-ui 使用了 go-sqlite3，需要 CGO 支持
# 如果交叉编译到 Linux，需要安装交叉编译工具链
# 推荐在目标服务器上编译，或者使用 CGO_ENABLED=1
export CGO_ENABLED=1

# 对于 Linux 目标平台，设置 C 交叉编译器
if [ "$BUILD_OS" = "linux" ] && [ "$(go env GOOS)" != "linux" ]; then
    # 交叉编译到 Linux，需要设置 CC
    # macOS 编译到 Linux 需要安装交叉编译工具
    if [ "$(go env GOOS)" = "darwin" ]; then
        # macOS 需要安装交叉编译工具
        if command -v x86_64-linux-musl-gcc > /dev/null 2>&1; then
            export CC=x86_64-linux-musl-gcc
            export CGO_ENABLED=1
        else
            print_warning "检测到交叉编译到 Linux，但未找到交叉编译工具"
            print_info "选项 1: 安装交叉编译工具（推荐使用 musl-cross）"
            print_info "选项 2: 在服务器上直接编译"
            print_info "选项 3: 使用 Docker 编译"
            print_warning "当前尝试继续编译，如果失败请在服务器上编译"
        fi
    fi
fi

print_info "编译环境: GOOS=$GOOS, GOARCH=$GOARCH, CGO_ENABLED=$CGO_ENABLED"
go build -ldflags="-s -w" -o "$XUI_BINARY_NAME" ./
if [ $? -eq 0 ]; then
    print_success "x-ui 编译成功"
else
    print_error "x-ui 编译失败"
    exit 1
fi

# 2. 编译 backend-proxy
print_info "步骤 2/6: 编译 backend-proxy 服务..."
cd backend-proxy

# backend-proxy 也使用 sqlite，需要 CGO
export CGO_ENABLED=1
if [ "$BUILD_OS" = "linux" ] && [ "$(go env GOOS)" != "linux" ]; then
    if [ "$(go env GOOS)" = "darwin" ] && command -v x86_64-linux-musl-gcc > /dev/null 2>&1; then
        export CC=x86_64-linux-musl-gcc
    fi
fi

go build -ldflags="-s -w" -o "$BACKEND_PROXY_BINARY_NAME" ./
if [ $? -eq 0 ]; then
    print_success "backend-proxy 编译成功"
else
    print_error "backend-proxy 编译失败"
    exit 1
fi

cd ..

# 3. 测试连接
print_info "步骤 3/6: 测试远程服务器连接..."
if $SSH_CMD "echo '连接成功'" > /dev/null 2>&1; then
    print_success "远程服务器连接正常"
else
    print_error "无法连接到远程服务器"
    print_info "请检查:"
    echo "  1. 服务器地址是否正确: $REMOTE_HOST"
    echo "  2. SSH 端口是否正确: ${REMOTE_PORT:-22}"
    echo "  3. SSH 密钥是否正确（如果使用）"
    echo "  4. 服务器是否允许 SSH 连接"
    exit 1
fi

# 4. 创建远程目录
print_info "步骤 4/6: 创建远程部署目录..."
$SSH_CMD "sudo mkdir -p $XUI_DEPLOY_PATH $BACKEND_PROXY_DEPLOY_PATH" || true
$SSH_CMD "sudo chown -R $REMOTE_USER:$REMOTE_USER $XUI_DEPLOY_PATH $BACKEND_PROXY_DEPLOY_PATH" || true
print_success "远程目录创建完成"

# 5. 上传文件
print_info "步骤 5/6: 上传文件到远程服务器..."

# 上传 x-ui（先上传到临时目录，再移动到目标位置）
print_info "上传 x-ui..."
TEMP_DIR="/tmp/x-ui-deploy-$$"
$SSH_CMD "mkdir -p $TEMP_DIR"
$SCP_CMD "$XUI_BINARY_NAME" "${REMOTE_USER}@${REMOTE_HOST}:${TEMP_DIR}/"
$SSH_CMD "sudo mv ${TEMP_DIR}/${XUI_BINARY_NAME} ${XUI_DEPLOY_PATH}/ && sudo chmod +x ${XUI_DEPLOY_PATH}/${XUI_BINARY_NAME} && sudo chown $REMOTE_USER:$REMOTE_USER ${XUI_DEPLOY_PATH}/${XUI_BINARY_NAME}"
$SSH_CMD "rm -rf $TEMP_DIR" || true
print_success "x-ui 上传完成"

# 上传 backend-proxy
print_info "上传 backend-proxy..."
$SSH_CMD "mkdir -p $TEMP_DIR"
$SCP_CMD "backend-proxy/${BACKEND_PROXY_BINARY_NAME}" "${REMOTE_USER}@${REMOTE_HOST}:${TEMP_DIR}/"
$SSH_CMD "sudo mv ${TEMP_DIR}/${BACKEND_PROXY_BINARY_NAME} ${BACKEND_PROXY_DEPLOY_PATH}/ && sudo chmod +x ${BACKEND_PROXY_DEPLOY_PATH}/${BACKEND_PROXY_BINARY_NAME} && sudo chown $REMOTE_USER:$REMOTE_USER ${BACKEND_PROXY_DEPLOY_PATH}/${BACKEND_PROXY_BINARY_NAME}"

# 上传 backend-proxy 配置文件示例（如果不存在）
$SCP_CMD "backend-proxy/env.example" "${REMOTE_USER}@${REMOTE_HOST}:${TEMP_DIR}/" || true
$SSH_CMD "sudo mv ${TEMP_DIR}/env.example ${BACKEND_PROXY_DEPLOY_PATH}/ && sudo chown $REMOTE_USER:$REMOTE_USER ${BACKEND_PROXY_DEPLOY_PATH}/env.example" || true
$SSH_CMD "rm -rf $TEMP_DIR" || true
print_success "backend-proxy 上传完成"

# 6. 创建 systemd 服务（如果启用）
if [ "$CREATE_SYSTEMD_SERVICE" = "yes" ]; then
    print_info "步骤 6/6: 创建 systemd 服务..."
    
    # 创建 x-ui 服务（先写入临时文件，然后用 sudo 移动）
    print_info "创建 x-ui systemd 服务..."
    TEMP_SERVICE="/tmp/${XUI_SERVICE_NAME}.service.$$"
    $SSH_CMD "cat > $TEMP_SERVICE << 'EOFSERVICE'
[Unit]
Description=x-ui Service
After=network.target

[Service]
Type=simple
WorkingDirectory=${XUI_DEPLOY_PATH}
ExecStart=${XUI_DEPLOY_PATH}/${XUI_BINARY_NAME}
Restart=always
RestartSec=5s
User=root

[Install]
WantedBy=multi-user.target
EOFSERVICE
    " && \
    $SSH_CMD "sudo mv $TEMP_SERVICE /etc/systemd/system/${XUI_SERVICE_NAME}.service && sudo chmod 644 /etc/systemd/system/${XUI_SERVICE_NAME}.service" && \
    print_success "x-ui 服务创建成功" || \
    print_warning "创建 x-ui 服务失败（可能没有权限）"
    
    # 创建 backend-proxy 服务
    print_info "创建 backend-proxy systemd 服务..."
    TEMP_SERVICE_PROXY="/tmp/${BACKEND_PROXY_SERVICE_NAME}.service.$$"
    $SSH_CMD "cat > $TEMP_SERVICE_PROXY << 'EOFSERVICE'
[Unit]
Description=Backend Proxy Service
After=network.target
Requires=network.target

[Service]
Type=simple
WorkingDirectory=${BACKEND_PROXY_DEPLOY_PATH}
ExecStart=${BACKEND_PROXY_DEPLOY_PATH}/${BACKEND_PROXY_BINARY_NAME}
Restart=always
RestartSec=5s
Environment=\"XUI_DB_PATH=/etc/x-ui/x-ui.db\"
Environment=\"XUI_SERVER_URL=http://localhost:54321\"
Environment=\"PROXY_PORT=${BACKEND_PROXY_PORT}\"

[Install]
WantedBy=multi-user.target
EOFSERVICE
    " && \
    $SSH_CMD "sudo mv $TEMP_SERVICE_PROXY /etc/systemd/system/${BACKEND_PROXY_SERVICE_NAME}.service && sudo chmod 644 /etc/systemd/system/${BACKEND_PROXY_SERVICE_NAME}.service" && \
    print_success "backend-proxy 服务创建成功" || \
    print_warning "创建 backend-proxy 服务失败（可能没有权限）"
    
    # 清理临时文件
    $SSH_CMD "rm -f $TEMP_SERVICE $TEMP_SERVICE_PROXY" || true
    
    # 重新加载 systemd（需要 sudo）
    print_info "重新加载 systemd..."
    if $SSH_CMD "sudo systemctl daemon-reload" 2>/dev/null; then
        print_success "systemd 重新加载成功"
    else
        print_warning "systemd 重新加载失败（可能没有权限，请手动执行: sudo systemctl daemon-reload）"
    fi
    
    print_info "systemd 服务创建完成"
    print_info "使用以下命令管理服务:"
    echo "  启动: systemctl start ${XUI_SERVICE_NAME} ${BACKEND_PROXY_SERVICE_NAME}"
    echo "  停止: systemctl stop ${XUI_SERVICE_NAME} ${BACKEND_PROXY_SERVICE_NAME}"
    echo "  状态: systemctl status ${XUI_SERVICE_NAME} ${BACKEND_PROXY_SERVICE_NAME}"
    echo "  开机自启: systemctl enable ${XUI_SERVICE_NAME} ${BACKEND_PROXY_SERVICE_NAME}"
else
    print_info "步骤 6/6: 跳过创建 systemd 服务"
fi

# 清理本地编译文件（可选）
read -p "是否删除本地编译文件？(y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    rm -f "$XUI_BINARY_NAME"
    rm -f "backend-proxy/${BACKEND_PROXY_BINARY_NAME}"
    print_success "本地编译文件已清理"
fi

echo ""
echo "=========================================="
print_success "部署完成！"
echo "=========================================="
echo ""
print_info "下一步操作:"
echo "1. SSH 登录服务器: $SSH_CMD"
echo "2. 配置 backend-proxy 环境变量:"
echo "   cd $BACKEND_PROXY_DEPLOY_PATH"
echo "   cp env.example .env"
echo "   nano .env  # 编辑配置文件"
echo ""
if [ "$CREATE_SYSTEMD_SERVICE" = "yes" ]; then
    echo "3. 启动服务:"
    echo "   systemctl start ${XUI_SERVICE_NAME}"
    echo "   systemctl start ${BACKEND_PROXY_SERVICE_NAME}"
    echo ""
    echo "4. 设置开机自启:"
    echo "   systemctl enable ${XUI_SERVICE_NAME}"
    echo "   systemctl enable ${BACKEND_PROXY_SERVICE_NAME}"
else
    echo "3. 启动服务（手动方式）:"
    echo "   cd $XUI_DEPLOY_PATH && ./${XUI_BINARY_NAME} &"
    echo "   cd $BACKEND_PROXY_DEPLOY_PATH && ./${BACKEND_PROXY_BINARY_NAME} &"
fi
echo ""

