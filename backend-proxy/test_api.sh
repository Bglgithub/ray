#!/bin/bash

# 测试后端代理服务接口的脚本

BASE_URL="${1:-http://localhost:8080}"

echo "=========================================="
echo "测试后端代理服务接口"
echo "服务器地址: $BASE_URL"
echo "=========================================="
echo ""

# 1. 健康检查
echo "1. 测试健康检查接口..."
curl -s "$BASE_URL/health" | jq . || curl -s "$BASE_URL/health"
echo -e "\n"

# 2. 测试创建入站（需要有效的订单）
echo "2. 测试创建入站接口..."
echo "注意：需要先创建订单并标记为paid状态"
echo "现在只需要传递订单ID和用户ID，其他参数由后端自动设置"
curl -X POST "$BASE_URL/api/v1/inbound/create" \
  -H "Content-Type: application/json" \
  -d '{
    "orderId": "ORDER_TEST_001",
    "userId": "USER_TEST_001"
  }' | jq . || curl -X POST "$BASE_URL/api/v1/inbound/create" \
  -H "Content-Type: application/json" \
  -d '{
    "orderId": "ORDER_TEST_001",
    "userId": "USER_TEST_001"
  }'
echo -e "\n\n"

# 3. 测试查询订单状态
echo "3. 测试查询订单状态接口..."
curl -X POST "$BASE_URL/api/v1/order/status" \
  -H "Content-Type: application/json" \
  -d '{
    "orderId": "ORDER_TEST_001"
  }' | jq . || curl -X POST "$BASE_URL/api/v1/order/status" \
  -H "Content-Type: application/json" \
  -d '{
    "orderId": "ORDER_TEST_001"
  }'
echo -e "\n"

echo "=========================================="
echo "测试完成"
echo "=========================================="

