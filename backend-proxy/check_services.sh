#!/bin/bash

# 快速检查服务状态脚本
# 使用方法: ./check_services.sh

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_info() {
    echo -e "${YELLOW}ℹ️  $1${NC}"
}

print_header() {
    echo -e "${BLUE}$1${NC}"
}

# 默认配置
BACKEND_PROXY_URL="${BACKEND_PROXY_URL:-http://localhost:8080}"
XUI_URL="${XUI_SERVER_URL:-http://localhost:54321}"

# 从 .env 文件读取配置（如果存在）
if [ -f ".env" ]; then
    ENV_XUI_URL=$(grep "^XUI_SERVER_URL=" .env 2>/dev/null | cut -d '=' -f2 | tr -d '"' | tr -d "'" | tr -d ' ')
    if [ -n "$ENV_XUI_URL" ]; then
        XUI_URL="$ENV_XUI_URL"
    fi
fi

echo "=========================================="
print_header "服务状态检查"
echo "=========================================="
echo ""

# 检查 backend-proxy
print_info "1. 检查 backend-proxy..."
print_info "   地址: $BACKEND_PROXY_URL"
if curl -s -f --max-time 3 "$BACKEND_PROXY_URL/health" > /dev/null 2>&1; then
    HEALTH_RESPONSE=$(curl -s "$BACKEND_PROXY_URL/health")
    print_success "backend-proxy 运行正常"
    echo "   响应: $HEALTH_RESPONSE"
else
    print_error "backend-proxy 未运行"
    echo ""
    echo "   启动命令:"
    echo "   cd $(pwd)"
    echo "   ./backend-proxy"
    echo ""
    BACKEND_OK=false
fi

echo ""

# 检查 x-ui
print_info "2. 检查 x-ui..."
print_info "   地址: $XUI_URL"
if curl -s -f --max-time 3 "$XUI_URL/login" > /dev/null 2>&1; then
    print_success "x-ui 运行正常"
elif curl -s -f --max-time 3 "$XUI_URL/api/v1/inbound/create" -X POST > /dev/null 2>&1; then
    print_success "x-ui API 可访问"
else
    print_error "x-ui 未运行或无法访问"
    echo ""
    echo "   启动命令:"
    echo "   cd .."
    echo "   ./x-ui"
    echo ""
    echo "   或者检查 x-ui 是否在其他地址"
    echo "   当前检查地址: $XUI_URL"
    echo "   可在 .env 文件中设置 XUI_SERVER_URL"
    echo ""
    XUI_OK=false
fi

echo ""

# 检查数据库文件
print_info "3. 检查数据库文件..."
DB_PATH="${XUI_DB_PATH:-/etc/x-ui/x-ui.db}"
if [ -f ".env" ]; then
    ENV_DB_PATH=$(grep "^XUI_DB_PATH=" .env 2>/dev/null | cut -d '=' -f2 | tr -d '"' | tr -d "'" | tr -d ' ')
    if [ -n "$ENV_DB_PATH" ]; then
        DB_PATH="$ENV_DB_PATH"
    fi
fi

print_info "   路径: $DB_PATH"
if [ -f "$DB_PATH" ]; then
    DB_SIZE=$(du -h "$DB_PATH" | cut -f1)
    print_success "数据库文件存在 (大小: $DB_SIZE)"
else
    print_error "数据库文件不存在"
    echo "   请确认:"
    echo "   1. x-ui 服务已启动并初始化数据库"
    echo "   2. 数据库路径配置正确: $DB_PATH"
    echo "   可在 .env 文件中设置 XUI_DB_PATH"
    echo ""
    DB_OK=false
fi

echo ""
echo "=========================================="

# 总结
if [ "${BACKEND_OK:-true}" = "true" ] && [ "${XUI_OK:-true}" = "true" ] && [ "${DB_OK:-true}" = "true" ]; then
    print_success "所有服务运行正常，可以开始测试！"
    echo ""
    echo "运行测试:"
    echo "  ./test_create_inbound.sh"
    exit 0
else
    print_error "部分服务未就绪，请先启动相关服务"
    exit 1
fi

