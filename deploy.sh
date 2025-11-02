#!/bin/bash

# x-ui 和 backend-proxy 一键部署脚本
# 使用方法: ./deploy.sh
#
# 注意：从 macOS 交叉编译到 Linux 比较复杂，特别是使用 CGO 时
# 推荐使用服务器端部署脚本: remote-build-deploy.sh

set -e  # 遇到错误立即退出

# 确保 PATH 包含常见的 Homebrew 路径（用于查找交叉编译工具）
export PATH="/opt/homebrew/bin:/usr/local/bin:$PATH"

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
REMOTE_HOST="your-server-ip-or-domain"
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
SSH_KEY=""  # 如果使用密钥，填写密钥路径，如: ~/.ssh/id_rsa
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

# 检查是否从 macOS 交叉编译到 Linux
if [ "$BUILD_OS" = "linux" ] && [ "$(go env GOOS 2>/dev/null)" = "darwin" ]; then
    echo ""
    print_warning "⚠️  检测到从 macOS 交叉编译到 Linux"
    echo ""
    print_info "交叉编译使用 CGO 时可能会遇到问题："
    echo "  - 需要 Linux 交叉编译工具链"
    echo "  - 需要正确配置 CGO 环境变量"
    echo "  - 可能存在兼容性问题"
    echo ""
    print_info "推荐方案：使用服务器端部署脚本（已在服务器上）"
    echo ""
    print_success "服务器端脚本位置: ~/remote-build-deploy.sh"
    print_success "服务器地址: ${REMOTE_HOST}"
    echo ""
    read -p "是否继续尝试本地交叉编译？(y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_info "已取消。推荐使用服务器端脚本："
        echo "  ssh ${REMOTE_USER}@${REMOTE_HOST}"
        echo "  ~/remote-build-deploy.sh"
        exit 0
    fi
    echo ""
fi

# 1. 编译 x-ui
print_info "步骤 1/6: 编译 x-ui 服务..."
cd "$(dirname "$0")"
if [ ! -d ".git" ]; then
    print_error "不在项目根目录"
    exit 1
fi

export GOOS="$BUILD_OS"
export GOARCH="$BUILD_ARCH"

# 检测并设置交叉编译工具
CC_FOUND=""

if [ "$BUILD_OS" = "linux" ] && [ "$(go env GOOS)" != "linux" ]; then
    # 交叉编译到 Linux
    if [ "$(go env GOOS)" = "darwin" ]; then
        # macOS 需要 Linux 交叉编译工具
        print_info "检测交叉编译工具..."
        
        # 尝试多种方法查找 musl-cross
        for test_path in \
            "$(command -v x86_64-linux-musl-gcc 2>/dev/null)" \
            "/opt/homebrew/bin/x86_64-linux-musl-gcc" \
            "/usr/local/bin/x86_64-linux-musl-gcc" \
            "$(which x86_64-linux-musl-gcc 2>/dev/null)"
        do
            if [ -n "$test_path" ] && [ -x "$test_path" ]; then
                MUSL_GCC="$test_path"
                print_info "找到 musl-cross: $MUSL_GCC"
                
                # 验证编译器
                if "$MUSL_GCC" --version > /dev/null 2>&1; then
                    CC_FOUND="$MUSL_GCC"
                    break
                fi
            fi
        done
        
        # 如果 musl 没找到，尝试 GNU 工具链
        if [ -z "$CC_FOUND" ]; then
            for test_path in \
                "$(command -v x86_64-linux-gnu-gcc 2>/dev/null)" \
                "/usr/bin/x86_64-linux-gnu-gcc"
            do
                if [ -n "$test_path" ] && [ -x "$test_path" ]; then
                    if "$test_path" --version > /dev/null 2>&1; then
                        CC_FOUND="$test_path"
                        break
                    fi
                fi
            done
        fi
        
        # 设置编译器
        if [ -n "$CC_FOUND" ]; then
            export CC="$CC_FOUND"
            export CGO_ENABLED=1
            
            if [[ "$CC_FOUND" == *"musl"* ]]; then
                # musl 交叉编译需要更多配置
                export CGO_LDFLAGS="-static"
                # 尝试找到 musl 的头文件和库路径
                MUSL_PREFIX=$(dirname $(dirname "$CC_FOUND"))
                if [ -d "$MUSL_PREFIX/x86_64-linux-musl" ]; then
                    export CGO_CFLAGS="-I$MUSL_PREFIX/x86_64-linux-musl/include"
                    export CGO_LDFLAGS="-L$MUSL_PREFIX/x86_64-linux-musl/lib -static"
                    print_info "设置 musl 头文件路径: $MUSL_PREFIX/x86_64-linux-musl"
                fi
                print_success "使用 musl-cross 交叉编译工具: $CC"
            else
                print_success "使用 GNU 交叉编译工具: $CC"
            fi
            
            # 验证编译器
            print_info "验证编译器..."
            if "$CC" --version > /dev/null 2>&1; then
                COMPILER_VERSION=$("$CC" --version 2>&1 | head -1)
                print_success "编译器验证成功: $COMPILER_VERSION"
            else
                print_error "编译器验证失败: $CC"
                exit 1
            fi
        else
            print_error "未找到可用的交叉编译工具"
            echo ""
            print_info "解决方案："
            echo "  方案 1（强烈推荐）: 使用服务器端部署脚本"
            echo "    已上传 remote-build-deploy.sh 到服务器: ${REMOTE_HOST}"
            echo "    SSH 登录后执行: ~/remote-build-deploy.sh"
            echo ""
            echo "  方案 2: 安装 musl-cross"
            echo "    brew install filosottile/musl-cross/musl-cross"
            echo "    然后重新运行此脚本"
            echo ""
            print_error "当前无法继续编译，请选择上述方案之一"
            exit 1
        fi
    fi
else
    # 同平台编译，直接启用 CGO
    export CGO_ENABLED=1
fi

print_info "编译环境: GOOS=$GOOS, GOARCH=$GOARCH, CGO_ENABLED=$CGO_ENABLED"
if [ -n "$CC" ]; then
    print_info "C 编译器: $CC"
    if [ -n "$CGO_CFLAGS" ]; then
        print_info "CGO_CFLAGS: $CGO_CFLAGS"
    fi
    if [ -n "$CGO_LDFLAGS" ]; then
        print_info "CGO_LDFLAGS: $CGO_LDFLAGS"
    fi
else
    # 对于非交叉编译，不需要检查
    if [ "$BUILD_OS" = "linux" ] && [ "$(go env GOOS)" != "linux" ]; then
        print_error "未设置 C 编译器！"
        print_error "从 macOS 交叉编译到 Linux 必须设置 CC 环境变量"
        exit 1
    fi
fi

print_info "开始编译..."
go build -ldflags="-s -w" -o "$XUI_BINARY_NAME" ./
if [ $? -eq 0 ]; then
    print_success "x-ui 编译成功"
else
    print_error "x-ui 编译失败"
    print_warning "如果是因为交叉编译问题，推荐使用服务器端脚本"
    print_info "执行: ssh ${REMOTE_USER}@${REMOTE_HOST} && ~/remote-build-deploy.sh"
    exit 1
fi

# 2. 编译 backend-proxy
print_info "步骤 2/6: 编译 backend-proxy 服务..."
cd backend-proxy

# backend-proxy 也使用 sqlite，需要 CGO
# 复用之前设置的 CC 和 CGO_ENABLED（在父 shell 中已经设置）
# 如果之前没有设置 CC，尝试查找
if [ -z "$CC" ] && [ "$BUILD_OS" = "linux" ] && [ "$(go env GOOS)" = "darwin" ]; then
    MUSL_GCC=""
    if command -v x86_64-linux-musl-gcc > /dev/null 2>&1; then
        MUSL_GCC=$(command -v x86_64-linux-musl-gcc)
    elif [ -f "/opt/homebrew/bin/x86_64-linux-musl-gcc" ]; then
        MUSL_GCC="/opt/homebrew/bin/x86_64-linux-musl-gcc"
    elif [ -f "/usr/local/bin/x86_64-linux-musl-gcc" ]; then
        MUSL_GCC="/usr/local/bin/x86_64-linux-musl-gcc"
    fi
    
    if [ -n "$MUSL_GCC" ]; then
        export CC="$MUSL_GCC"
        export CGO_ENABLED=1
        export CGO_LDFLAGS="-static"
        MUSL_PREFIX=$(dirname $(dirname "$MUSL_GCC"))
        if [ -d "$MUSL_PREFIX/x86_64-linux-musl" ]; then
            export CGO_CFLAGS="-I$MUSL_PREFIX/x86_64-linux-musl/include"
            export CGO_LDFLAGS="-L$MUSL_PREFIX/x86_64-linux-musl/lib -static"
        fi
    elif command -v x86_64-linux-gnu-gcc > /dev/null 2>&1; then
        export CC=$(command -v x86_64-linux-gnu-gcc)
        export CGO_ENABLED=1
    else
        export CGO_ENABLED=1
    fi
else
    export CGO_ENABLED=1
fi

if [ -n "$CC" ]; then
    print_info "backend-proxy 使用 C 编译器: $CC"
fi

go build -ldflags="-s -w" -o "$BACKEND_PROXY_BINARY_NAME" ./
if [ $? -eq 0 ]; then
    print_success "backend-proxy 编译成功"
else
    print_error "backend-proxy 编译失败"
    print_warning "如果是因为交叉编译问题，推荐使用服务器端脚本"
    print_info "执行: ssh ${REMOTE_USER}@${REMOTE_HOST} && ~/remote-build-deploy.sh"
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
print_info "💡 提示: 如果遇到交叉编译问题，使用服务器端脚本更简单："
echo "   ssh ${REMOTE_USER}@${REMOTE_HOST}"
echo "   ~/remote-build-deploy.sh"
echo ""
