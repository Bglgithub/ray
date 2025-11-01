# API 接口变更说明

## 变更内容

为了简化客户端调用，`/api/v1/inbound/create` 接口的请求参数已简化。

## 变更前

**请求参数**（需要传递所有参数）：
```json
{
  "orderId": "ORDER_123456789",
  "userId": "USER_12345",
  "protocol": "vmess",
  "port": 443,
  "remark": "用户节点",
  "expiryTime": 1735689600000,
  "total": 10737418240,
  "settings": "{}",
  "streamSettings": "{}",
  "sniffing": "{}",
  "listen": ""
}
```

## 变更后

**请求参数**（只需要订单ID和用户ID）：
```json
{
  "orderId": "ORDER_123456789",
  "userId": "USER_12345"
}
```

## 后端自动设置的默认值

以下参数由后端自动设置，客户端无需传递：

### 协议相关
- **Protocol**: 使用环境变量 `DEFAULT_PROTOCOL`（默认：`vmess`）
- **Settings**: 自动生成默认配置（根据协议类型生成 UUID、密码等）
- **StreamSettings**: 使用默认传输配置
- **Sniffing**: 使用默认流量嗅探配置

### 端口和网络
- **Port**: 自动分配可用端口（从 10000 开始查找）
- **Listen**: 默认为空字符串（使用 x-ui 默认监听地址）

### 备注和限制
- **Remark**: 自动生成（格式：`用户节点-{订单号后4位}`）
- **ExpiryTime**: 从订单中读取（如果订单有 `expiresAt`），否则为 0（永不过期）
- **Total**: 从订单中读取（如果订单有流量限制），否则为 0（无限制）

## 配置说明

可以通过环境变量配置默认协议：

```env
# .env 文件
DEFAULT_PROTOCOL=vmess  # 可选值：vmess/vless/trojan/shadowsocks
```

## 客户端调用示例

### Kotlin/Retrofit

```kotlin
// 数据模型（简化版）
data class CreateInboundRequest(
    val orderId: String,
    val userId: String
)

// API 接口定义
interface XUIApiService {
    @POST("/api/v1/inbound/create")
    fun createInbound(
        @Body request: CreateInboundRequest
    ): Call<ApiResponse<CreateInboundResponse>>
}

// 使用
val request = CreateInboundRequest(
    orderId = "ORDER_123456789",
    userId = "USER_12345"
)

apiService.createInbound(request).enqueue(...)
```

### curl 示例

```bash
curl -X POST http://localhost:8080/api/v1/inbound/create \
  -H "Content-Type: application/json" \
  -d '{
    "orderId": "ORDER_123456789",
    "userId": "USER_12345"
  }'
```

## 向后兼容

如果客户端传递了其他参数（如 `protocol`、`port` 等），这些参数会被忽略，使用后端默认值。

## 优势

1. ✅ **简化客户端**：只需传递订单ID和用户ID
2. ✅ **统一配置**：所有默认配置在后端集中管理
3. ✅ **易于维护**：修改默认值无需更新客户端
4. ✅ **减少错误**：客户端无法传递错误配置

