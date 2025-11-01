# 验证 `/api/v1/inbound/create` 接口测试指南

## 测试前准备

### 1. 确保服务运行

**启动 x-ui 服务**：
```bash
# 在 x-ui 目录下
cd /Users/guolinbao/projects/x-ui/x-ui
./x-ui
```

**启动 backend-proxy 服务**：
```bash
# 在 backend-proxy 目录下
cd /Users/guolinbao/projects/x-ui/x-ui/backend-proxy
./backend-proxy
```

### 2. 检查服务状态

**检查 backend-proxy 是否运行**：
```bash
curl http://localhost:8080/health
```

**预期响应**：
```json
{
  "success": true,
  "msg": "服务运行正常"
}
```

**检查 x-ui 是否运行**：
```bash
curl http://localhost:54321/login
```

应该返回登录页面（HTML）。

---

## 测试步骤

### 方法 1：使用测试脚本（推荐）

我们提供了一个自动化测试脚本，会：
1. 创建测试订单
2. 标记订单为已支付
3. 调用创建接口
4. 验证结果

```bash
cd /Users/guolinbao/projects/x-ui/x-ui/backend-proxy
chmod +x test_create_inbound.sh
./test_create_inbound.sh
```

### 方法 2：手动测试（curl）

#### 步骤 1：准备测试订单

首先需要创建一个订单并标记为已支付。可以通过 SQLite 命令行：

```bash
sqlite3 /etc/x-ui/x-ui.db << EOF
-- 创建测试订单
INSERT OR REPLACE INTO orders (order_id, user_id, status, amount, expires_at, created_at)
VALUES (
  'TEST_ORDER_001',
  'TEST_USER_001',
  'paid',  -- 必须标记为 paid
  10000,   -- 金额（分）
  0,       -- 0 表示永不过期
  strftime('%s', 'now')
);

-- 更新支付时间
UPDATE orders SET paid_at = strftime('%s', 'now') * 1000 WHERE order_id = 'TEST_ORDER_001';

-- 验证订单
SELECT * FROM orders WHERE order_id = 'TEST_ORDER_001';
EOF
```

#### 步骤 2：调用创建接口

```bash
curl -X POST http://localhost:8080/api/v1/inbound/create \
  -H "Content-Type: application/json" \
  -d '{
    "orderId": "TEST_ORDER_001",
    "userId": "TEST_USER_001"
  }' | jq .
```

**预期成功响应**：
```json
{
  "success": true,
  "msg": "创建入站配置成功",
  "data": {
    "inboundId": 1,
    "port": 12345,
    "tag": "inbound-12345"
  }
}
```

**如果失败，可能的错误**：

1. **订单不存在**：
```json
{
  "success": false,
  "msg": "创建入站配置失败: 订单不存在"
}
```

2. **订单未支付**：
```json
{
  "success": false,
  "msg": "创建入站配置失败: 订单尚未支付"
}
```

3. **订单已被使用**：
```json
{
  "success": false,
  "msg": "创建入站配置失败: 订单已被使用"
}
```

### 方法 3：使用 Postman 或 HTTPie

#### Postman 配置

1. **方法**：POST
2. **URL**：`http://localhost:8080/api/v1/inbound/create`
3. **Headers**：
   - `Content-Type: application/json`
4. **Body** (raw JSON)：
```json
{
  "orderId": "TEST_ORDER_001",
  "userId": "TEST_USER_001"
}
```

#### HTTPie 命令

```bash
http POST http://localhost:8080/api/v1/inbound/create \
  orderId=TEST_ORDER_001 \
  userId=TEST_USER_001
```

---

## 验证接口调用成功的标准

### 1. HTTP 状态码

- ✅ **成功**：`200 OK`
- ❌ **失败**：`400 Bad Request`（参数错误）、`500 Internal Server Error`（服务器错误）

### 2. 响应体结构

**成功时**：
```json
{
  "success": true,
  "msg": "创建入站配置成功",
  "data": {
    "inboundId": 1,      // 入站ID
    "port": 12345,       // 端口号
    "tag": "inbound-12345" // 标签
  }
}
```

**失败时**：
```json
{
  "success": false,
  "msg": "错误原因"
}
```

### 3. 数据库验证

验证订单状态已更新：
```bash
sqlite3 /etc/x-ui/x-ui.db "SELECT * FROM orders WHERE order_id = 'TEST_ORDER_001';"
```

应该看到：
- `status = 'used'`（已使用）
- `inbound_id` 有值（关联的入站ID）
- `used_at` 有值（使用时间）

验证入站已创建：
```bash
sqlite3 /etc/x-ui/x-ui.db "SELECT id, port, protocol, remark FROM inbounds WHERE id = (SELECT inbound_id FROM orders WHERE order_id = 'TEST_ORDER_001');"
```

### 4. x-ui 管理面板验证

访问 `http://localhost:54321`，登录后查看入站列表，应该能看到新创建的入站配置。

### 5. 日志验证

**backend-proxy 日志**：
```bash
# 应该看到类似日志
[backend-proxy] 调用x-ui API成功
```

**x-ui 日志**：
查看 x-ui 服务日志，应该看到创建入站的日志。

---

## 完整测试流程示例

```bash
#!/bin/bash

# 完整的测试流程
BASE_URL="http://localhost:8080"
ORDER_ID="TEST_ORDER_$(date +%s)"  # 使用时间戳确保唯一
USER_ID="TEST_USER_001"

echo "=========================================="
echo "测试创建入站接口"
echo "=========================================="
echo "订单ID: $ORDER_ID"
echo "用户ID: $USER_ID"
echo ""

# 1. 创建并支付订单
echo "1. 创建测试订单..."
sqlite3 /etc/x-ui/x-ui.db << EOF
INSERT OR REPLACE INTO orders (order_id, user_id, status, amount, expires_at, created_at, paid_at)
VALUES (
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
    echo "✅ 订单创建成功"
else
    echo "❌ 订单创建失败"
    exit 1
fi

# 2. 调用创建接口
echo ""
echo "2. 调用创建入站接口..."
RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/inbound/create" \
  -H "Content-Type: application/json" \
  -d "{
    \"orderId\": \"$ORDER_ID\",
    \"userId\": \"$USER_ID\"
  }")

echo "响应: $RESPONSE"
echo ""

# 3. 解析响应
SUCCESS=$(echo $RESPONSE | jq -r '.success')
MSG=$(echo $RESPONSE | jq -r '.msg')

if [ "$SUCCESS" = "true" ]; then
    echo "✅ 接口调用成功"
    INBOUND_ID=$(echo $RESPONSE | jq -r '.data.inboundId')
    PORT=$(echo $RESPONSE | jq -r '.data.port')
    echo "   入站ID: $INBOUND_ID"
    echo "   端口: $PORT"
else
    echo "❌ 接口调用失败: $MSG"
    exit 1
fi

# 4. 验证数据库
echo ""
echo "3. 验证数据库..."
ORDER_STATUS=$(sqlite3 /etc/x-ui/x-ui.db "SELECT status FROM orders WHERE order_id = '$ORDER_ID';")
INBOUND_ID_DB=$(sqlite3 /etc/x-ui/x-ui.db "SELECT inbound_id FROM orders WHERE order_id = '$ORDER_ID';")

if [ "$ORDER_STATUS" = "used" ] && [ -n "$INBOUND_ID_DB" ]; then
    echo "✅ 订单状态已更新为 'used'"
    echo "   关联入站ID: $INBOUND_ID_DB"
else
    echo "❌ 订单状态验证失败"
    exit 1
fi

echo ""
echo "=========================================="
echo "测试完成！"
echo "=========================================="
```

---

## 常见问题排查

### 问题 1：`订单不存在`

**原因**：订单还没有创建，或者订单号不正确。

**解决**：
1. 检查订单是否已创建
2. 确认订单号拼写正确

### 问题 2：`订单尚未支付`

**原因**：订单状态不是 `paid`。

**解决**：
```sql
UPDATE orders SET status = 'paid', paid_at = strftime('%s', 'now') * 1000 
WHERE order_id = 'TEST_ORDER_001';
```

### 问题 3：`订单已被使用`

**原因**：订单已经用来创建过入站了。

**解决**：使用新的订单号，或重置订单状态（仅测试用）：
```sql
UPDATE orders SET status = 'paid', used_at = 0, inbound_id = 0 
WHERE order_id = 'TEST_ORDER_001';
```

### 问题 4：`端口已被占用`

**原因**：自动分配的端口已被使用。

**解决**：x-ui 会自动选择下一个可用端口，通常不需要手动处理。

### 问题 5：`API Key 验证失败`

**原因**：backend-proxy 配置的 API Key 不正确。

**解决**：
1. 检查 `.env` 文件中的 `XUI_API_KEY` 和 `XUI_API_SECRET`
2. 确认 x-ui 数据库中存在对应的 API Key
3. 重启 backend-proxy 服务

### 问题 6：`x-ui 服务未响应`

**原因**：x-ui 服务没有运行，或地址配置错误。

**解决**：
1. 确认 x-ui 服务正在运行
2. 检查 `.env` 文件中的 `XUI_SERVER_URL` 是否正确
3. 测试 x-ui 服务是否可访问：`curl http://localhost:54321/health`

---

## 自动化测试脚本

使用提供的脚本可以一键完成所有测试：

```bash
cd /Users/guolinbao/projects/x-ui/x-ui/backend-proxy
./test_create_inbound.sh
```

脚本会自动：
- 创建测试订单
- 标记为已支付
- 调用接口
- 验证结果
- 清理测试数据（可选）

---

## 测试检查清单

- [ ] x-ui 服务运行正常
- [ ] backend-proxy 服务运行正常
- [ ] 健康检查接口返回成功
- [ ] 测试订单已创建并标记为 `paid`
- [ ] 接口调用返回 `200 OK`
- [ ] 响应体 `success` 为 `true`
- [ ] 返回了 `inboundId`、`port`、`tag`
- [ ] 订单状态已更新为 `used`
- [ ] 订单关联了 `inbound_id`
- [ ] 数据库中确实创建了入站记录
- [ ] x-ui 管理面板可以看到新入站

---

## 性能测试（可选）

如果需要测试接口性能：

```bash
# 使用 Apache Bench (ab)
ab -n 100 -c 10 -p test_request.json -T application/json \
   http://localhost:8080/api/v1/inbound/create
```

**注意**：需要准备足够的测试订单，每个订单只能使用一次。

---

## 总结

验证接口调用成功的**关键指标**：

1. ✅ **HTTP 状态码**：200 OK
2. ✅ **响应 success**：true
3. ✅ **返回数据**：包含 inboundId、port、tag
4. ✅ **数据库验证**：订单状态为 `used`，入站记录已创建
5. ✅ **x-ui 面板**：能看到新创建的入站配置

如果以上所有指标都满足，说明接口调用**完全成功**！

