#!/bin/bash

# 自动化测试脚本：创建入站接口
# 使用方法: ./test_create_inbound.sh

set -e  # 遇到错误立即退出

# 配置
BASE_URL="${1:-http://localhost:8080}"
DB_PATH="${XUI_DB_PATH:-/etc/x-ui/x-ui.db}"
ORDER_ID="TEST_ORDER_$(date +%s)"  # 使用时间戳确保唯一
USER_ID="TEST_USER_001"
CLEANUP="${2:-true}"  # 是否清理测试数据

# 从 .env 文件读取 x-ui 地址（如果存在）
if [ -f ".env" ]; then
    XUI_SERVER_URL=$(grep "^XUI_SERVER_URL=" .env | cut -d '=' -f2 | tr -d '"' | tr -d "'")
    if [ -n "$XUI_SERVER_URL" ]; then
        XUI_BASE_URL="$XUI_SERVER_URL"
    else
        XUI_BASE_URL="http://localhost:54321"
    fi
else
    XUI_BASE_URL="http://localhost:54321"
fi

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 打印函数
print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_info() {
    echo -e "${YELLOW}ℹ️  $1${NC}"
}

# 检查服务是否运行
check_service() {
    print_info "检查服务状态..."
    
    # 检查 backend-proxy
    print_info "检查 backend-proxy ($BASE_URL)..."
    if curl -s -f "$BASE_URL/health" > /dev/null 2>&1; then
        print_success "backend-proxy 服务运行正常 ($BASE_URL)"
    else
        print_error "backend-proxy 服务未运行！"
        print_info "请先启动 backend-proxy 服务："
        echo "  cd $(pwd)"
        echo "  ./backend-proxy"
        exit 1
    fi
    
    # 检查 x-ui（必需，因为 backend-proxy 需要调用 x-ui API）
    print_info "检查 x-ui ($XUI_BASE_URL)..."
    # 尝试访问 x-ui 的登录页面或 API
    if curl -s -f "$XUI_BASE_URL/login" > /dev/null 2>&1 || \
       curl -s -f "$XUI_BASE_URL/api/v1/inbound/create" -X POST > /dev/null 2>&1; then
        print_success "x-ui 服务运行正常 ($XUI_BASE_URL)"
    else
        print_error "x-ui 服务未运行或无法访问！"
        print_info "请先启动 x-ui 服务："
        echo "  cd ../"
        echo "  ./x-ui"
        print_info "或者检查 x-ui 是否在不同地址（当前检查: $XUI_BASE_URL）"
        print_info "可以在 .env 文件中设置 XUI_SERVER_URL 来配置 x-ui 地址"
        exit 1
    fi
}

# 检查数据库文件
check_database() {
    print_info "检查数据库文件..."
    
    if [ ! -f "$DB_PATH" ]; then
        print_error "数据库文件不存在: $DB_PATH"
        print_info "请检查 XUI_DB_PATH 环境变量或确认 x-ui 服务已初始化数据库"
        exit 1
    fi
    
    print_success "数据库文件存在: $DB_PATH"
}

# 创建测试订单
create_test_order() {
    print_info "创建测试订单: $ORDER_ID"
    
    sqlite3 "$DB_PATH" << EOF
INSERT OR REPLACE INTO orders (
    order_id, 
    user_id, 
    status, 
    amount, 
    expires_at, 
    created_at, 
    paid_at
) VALUES (
    '$ORDER_ID',
    '$USER_ID',
    'paid',
    10000,
    0,
    strftime('%s', 'now'),
    strftime('%s', 'now') * 1000
);
EOF
    
    if [ $? -eq 0 ]; then
        print_success "订单创建成功"
        
        # 验证订单
        ORDER_STATUS=$(sqlite3 "$DB_PATH" "SELECT status FROM orders WHERE order_id = '$ORDER_ID';")
        if [ "$ORDER_STATUS" = "paid" ]; then
            print_success "订单状态: $ORDER_STATUS"
        else
            print_error "订单状态不正确: $ORDER_STATUS"
            exit 1
        fi
    else
        print_error "订单创建失败"
        exit 1
    fi
}

# 调用创建接口
call_create_api() {
    print_info "调用创建入站接口..."
    
    # 使用临时文件存储响应（兼容 macOS 和 Linux）
    TEMP_FILE=$(mktemp)
    
    HTTP_CODE=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/inbound/create" \
      -H "Content-Type: application/json" \
      -d "{
        \"orderId\": \"$ORDER_ID\",
        \"userId\": \"$USER_ID\"
      }" -o "$TEMP_FILE")
    
    # 分离响应体和状态码（兼容 macOS）
    HTTP_CODE=$(echo "$HTTP_CODE" | tail -n 1)
    HTTP_BODY=$(cat "$TEMP_FILE")
    rm -f "$TEMP_FILE"
    
    echo ""
    echo "HTTP 状态码: $HTTP_CODE"
    echo "响应内容:"
    if command -v jq > /dev/null 2>&1; then
        echo "$HTTP_BODY" | jq . 2>/dev/null || echo "$HTTP_BODY"
    else
        echo "$HTTP_BODY"
    fi
    echo ""
    
    if [ "$HTTP_CODE" != "200" ]; then
        print_error "HTTP 状态码错误: $HTTP_CODE"
        return 1
    fi
    
    # 检查响应体
    if command -v jq > /dev/null 2>&1; then
        SUCCESS=$(echo "$HTTP_BODY" | jq -r '.success' 2>/dev/null)
        MSG=$(echo "$HTTP_BODY" | jq -r '.msg' 2>/dev/null)
    else
        # 如果没有 jq，使用 grep 和 sed 简单解析
        SUCCESS=$(echo "$HTTP_BODY" | grep -o '"success":[^,}]*' | cut -d':' -f2 | tr -d ' ')
        MSG=$(echo "$HTTP_BODY" | grep -o '"msg":"[^"]*"' | cut -d'"' -f4)
    fi
    
    if [ "$SUCCESS" != "true" ]; then
        print_error "接口调用失败: $MSG"
        return 1
    fi
    
    print_success "接口调用成功: $MSG"
    
    # 提取返回的数据
    if command -v jq > /dev/null 2>&1; then
        INBOUND_ID=$(echo "$HTTP_BODY" | jq -r '.data.inboundId' 2>/dev/null)
        PORT=$(echo "$HTTP_BODY" | jq -r '.data.port' 2>/dev/null)
        TAG=$(echo "$HTTP_BODY" | jq -r '.data.tag' 2>/dev/null)
    else
        INBOUND_ID=$(echo "$HTTP_BODY" | grep -o '"inboundId":[^,}]*' | cut -d':' -f2 | tr -d ' ')
        PORT=$(echo "$HTTP_BODY" | grep -o '"port":[^,}]*' | cut -d':' -f2 | tr -d ' ')
        TAG=$(echo "$HTTP_BODY" | grep -o '"tag":"[^"]*"' | cut -d'"' -f4)
    fi
    
    if [ -n "$INBOUND_ID" ] && [ "$INBOUND_ID" != "null" ] && [ "$INBOUND_ID" != "" ]; then
        echo "   入站ID: $INBOUND_ID"
        echo "   端口: $PORT"
        echo "   标签: $TAG"
        return 0
    else
        print_error "响应中缺少入站信息"
        return 1
    fi
}

# 验证数据库
verify_database() {
    print_info "验证数据库状态..."
    
    # 检查订单状态
    ORDER_STATUS=$(sqlite3 "$DB_PATH" "SELECT status FROM orders WHERE order_id = '$ORDER_ID';")
    INBOUND_ID_DB=$(sqlite3 "$DB_PATH" "SELECT inbound_id FROM orders WHERE order_id = '$ORDER_ID';")
    USED_AT=$(sqlite3 "$DB_PATH" "SELECT used_at FROM orders WHERE order_id = '$ORDER_ID';")
    
    if [ "$ORDER_STATUS" = "used" ]; then
        print_success "订单状态已更新为 'used'"
    else
        print_error "订单状态未更新: $ORDER_STATUS（期望: used）"
        return 1
    fi
    
    if [ -n "$INBOUND_ID_DB" ] && [ "$INBOUND_ID_DB" != "0" ]; then
        print_success "订单已关联入站ID: $INBOUND_ID_DB"
    else
        print_error "订单未关联入站ID"
        return 1
    fi
    
    if [ -n "$USED_AT" ] && [ "$USED_AT" != "0" ]; then
        print_success "使用时间已记录: $USED_AT"
    else
        print_error "使用时间未记录"
        return 1
    fi
    
    # 检查入站记录
    INBOUND_COUNT=$(sqlite3 "$DB_PATH" "SELECT COUNT(*) FROM inbounds WHERE id = $INBOUND_ID_DB;")
    
    if [ "$INBOUND_COUNT" = "1" ]; then
        print_success "入站记录已创建"
        
        # 显示入站详情
        INBOUND_INFO=$(sqlite3 "$DB_PATH" -json "SELECT id, port, protocol, remark, enable FROM inbounds WHERE id = $INBOUND_ID_DB;" | jq .)
        echo "   入站详情:"
        echo "$INBOUND_INFO" | jq .
    else
        print_error "入站记录未创建或ID不匹配"
        return 1
    fi
}

# 清理测试数据
cleanup_test_data() {
    if [ "$CLEANUP" = "true" ]; then
        print_info "清理测试数据..."
        
        # 删除测试订单
        sqlite3 "$DB_PATH" "DELETE FROM orders WHERE order_id = '$ORDER_ID';" 2>/dev/null
        
        # 如果有入站ID，删除入站记录（可选，根据需求决定是否删除）
        # INBOUND_ID_DB=$(sqlite3 "$DB_PATH" "SELECT inbound_id FROM orders WHERE order_id = '$ORDER_ID';" 2>/dev/null)
        # if [ -n "$INBOUND_ID_DB" ] && [ "$INBOUND_ID_DB" != "0" ]; then
        #     sqlite3 "$DB_PATH" "DELETE FROM inbounds WHERE id = $INBOUND_ID_DB;" 2>/dev/null
        # fi
        
        print_success "测试数据已清理"
    else
        print_info "保留测试数据（订单ID: $ORDER_ID）"
    fi
}

# 主函数
main() {
    echo "=========================================="
    echo "测试创建入站接口"
    echo "=========================================="
    echo "订单ID: $ORDER_ID"
    echo "用户ID: $USER_ID"
    echo "服务地址: $BASE_URL"
    echo "数据库路径: $DB_PATH"
    echo ""
    
    # 执行测试步骤
    check_service
    echo ""
    
    check_database
    echo ""
    
    create_test_order
    echo ""
    
    if call_create_api; then
        echo ""
        if verify_database; then
            echo ""
            print_success "所有测试通过！"
            echo ""
            cleanup_test_data
            echo ""
            echo "=========================================="
            echo "测试完成"
            echo "=========================================="
            exit 0
        else
            print_error "数据库验证失败"
            exit 1
        fi
    else
        print_error "接口调用失败"
        exit 1
    fi
}

# 运行主函数
main

