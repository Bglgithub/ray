# 测试接口说明

## 快速创建/更新订单接口

### 接口地址
```
GET /test/create-order
```

### 功能
快速创建或更新订单记录，方便测试使用。

### 参数说明

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `orderId` | string | ✅ 是 | 订单号（唯一标识） |
| `userId` | string | ✅ 是 | 用户ID |
| `expiresAt` | int64 | ❌ 否 | 过期时间（毫秒时间戳），默认为0表示永不过期 |

### 行为说明

- **订单不存在**：创建新订单，状态自动设为 `paid`（已支付），方便测试
- **订单已存在**：更新订单的 `userId` 和 `expiresAt`，如果状态不是 `paid` 也会更新为 `paid`

### 使用示例

#### 1. 浏览器访问

**创建新订单（永不过期）**：
```
http://localhost:8080/test/create-order?orderId=TEST_ORDER_001&userId=USER_001
```

**创建新订单（设置过期时间）**：
```
http://localhost:8080/test/create-order?orderId=TEST_ORDER_002&userId=USER_002&expiresAt=1735689600000
```

**更新已存在的订单**：
```
http://localhost:8080/test/create-order?orderId=TEST_ORDER_001&userId=USER_003&expiresAt=1735689600000
```

#### 2. 使用 curl 命令

```bash
# 创建订单（永不过期）
curl "http://localhost:8080/test/create-order?orderId=TEST_ORDER_001&userId=USER_001"

# 创建订单（30天后过期）
# 先计算30天后的时间戳（毫秒）
# expiresAt=$(($(date +%s) + 30*24*60*60))000
curl "http://localhost:8080/test/create-order?orderId=TEST_ORDER_002&userId=USER_002&expiresAt=1735689600000"

# 更新订单
curl "http://localhost:8080/test/create-order?orderId=TEST_ORDER_001&userId=USER_003&expiresAt=1735689600000"
```

#### 3. 时间戳计算

**JavaScript**：
```javascript
// 30天后的时间戳（毫秒）
const expiresAt = Date.now() + 30 * 24 * 60 * 60 * 1000;
console.log(expiresAt);

// 特定日期的时间戳
const date = new Date('2025-01-01');
const timestamp = date.getTime();
console.log(timestamp);
```

**在线工具**：
- https://www.epochconverter.com/ （Unix 时间戳转换）

### 响应示例

#### 成功创建订单
```json
{
  "success": true,
  "msg": "订单已创建: TEST_ORDER_001",
  "data": {
    "orderId": "TEST_ORDER_001",
    "userId": "USER_001",
    "expiresAt": 0,
    "status": "paid",
    "action": "created"
  }
}
```

#### 成功更新订单
```json
{
  "success": true,
  "msg": "订单已更新: TEST_ORDER_001",
  "data": {
    "orderId": "TEST_ORDER_001",
    "userId": "USER_003",
    "expiresAt": 1735689600000,
    "status": "paid",
    "action": "updated"
  }
}
```

#### 错误响应
```json
{
  "success": false,
  "msg": "参数错误: orderId 不能为空"
}
```

### 注意事项

1. **无需认证**：此接口为测试接口，不需要 API Key 认证，可直接访问
2. **自动设为已支付**：创建的订单状态自动设为 `paid`，方便直接测试创建入站接口
3. **过期时间**：
   - `0` 或空：表示永不过期
   - 时间戳单位：**毫秒**（不是秒）
4. **订单唯一性**：`orderId` 是唯一标识，如果已存在则更新，不存在则创建

### 完整测试流程

1. **创建测试订单**：
   ```
   http://localhost:8080/test/create-order?orderId=TEST_001&userId=USER_001
   ```

2. **验证订单状态**：
   ```bash
   curl -X POST http://localhost:8080/api/v1/order/status \
     -H "Content-Type: application/json" \
     -d '{"orderId": "TEST_001"}'
   ```

3. **创建入站配置**：
   ```bash
   curl -X POST http://localhost:8080/api/v1/inbound/create \
     -H "Content-Type: application/json" \
     -d '{"orderId": "TEST_001", "userId": "USER_001"}'
   ```

### 常用时间戳

| 描述 | 时间戳（毫秒） | URL 参数 |
|------|---------------|----------|
| 永不过期 | 0 | `expiresAt=0` 或不传 |
| 1天后 | `Date.now() + 86400000` | 动态计算 |
| 30天后 | `Date.now() + 2592000000` | 动态计算 |
| 2025-01-01 | `1735689600000` | `expiresAt=1735689600000` |
| 2026-01-01 | `1767225600000` | `expiresAt=1767225600000` |

---

**提示**：此接口仅用于开发测试，生产环境请谨慎使用或移除此接口。

