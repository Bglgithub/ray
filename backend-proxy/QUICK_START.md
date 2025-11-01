# 快速开始指南

## 🚀 快速编译和测试

### 1. 编译工程

```bash
cd backend-proxy
go build -o backend-proxy
```

### 2. 配置环境（如果还没有）

```bash
# 如果 .env 文件不存在，从示例复制
cp env.example .env

# 编辑 .env，至少确认数据库路径正确
# XUI_DB_PATH=/etc/x-ui/x-ui.db  （根据实际情况调整）
```

### 3. 启动服务

```bash
./backend-proxy
```

**首次启动时会看到：**
```
检测到API Key或Secret为空，尝试自动创建...
✅ 创建新的API Key: xui_abc123...
✅ 已更新.env文件中的API Key和Secret
✅ 自动创建API Key成功: xui_abc123...
后端代理服务启动在端口 8080
x-ui服务器地址: http://localhost:54321
```

### 4. 测试接口（新开一个终端）

#### 4.1 健康检查

```bash
curl http://localhost:8080/health
```

**预期输出：**
```json
{"success":true,"msg":"服务运行正常"}
```

#### 4.2 测试创建入站接口

**注意**：需要先创建测试订单（见下方）

```bash
curl -X POST http://localhost:8080/api/v1/inbound/create \
  -H "Content-Type: application/json" \
  -d '{
    "orderId": "ORDER_TEST_001",
    "userId": "USER_TEST_001",
    "protocol": "vmess",
    "remark": "测试节点"
  }'
```

#### 4.3 测试查询订单状态

```bash
curl -X POST http://localhost:8080/api/v1/order/status \
  -H "Content-Type: application/json" \
  -d '{
    "orderId": "ORDER_TEST_001"
  }'
```

## 📝 创建测试订单

在测试创建入站接口前，需要先在 x-ui 数据库中创建测试订单。

### 方法 1：使用 SQLite 命令行（最简单）

```bash
sqlite3 /etc/x-ui/x-ui.db << 'EOF'
INSERT INTO orders (order_id, user_id, status, amount, created_at, expires_at, paid_at)
VALUES (
  'ORDER_TEST_001', 
  'USER_TEST_001', 
  'paid', 
  10000, 
  (strftime('%s', 'now')) * 1000, 
  (strftime('%s', 'now', '+1 day')) * 1000,
  (strftime('%s', 'now')) * 1000
);
SELECT * FROM orders WHERE order_id = 'ORDER_TEST_001';
EOF
```

### 方法 2：使用 Go 脚本

在 x-ui 项目根目录创建 `create_test_order.go`：

```go
package main

import (
    "fmt"
    "time"
    "x-ui/database"
    "x-ui/database/model"
    "x-ui/config"
)

func main() {
    err := database.InitDB(config.GetDBPath())
    if err != nil {
        panic(err)
    }
    
    order := &model.Order{
        OrderId:   "ORDER_TEST_001",
        UserId:    "USER_TEST_001",
        Status:    "paid",
        Amount:    10000,
        PaidAt:    time.Now().Unix() * 1000,
        ExpiresAt: time.Now().Add(24 * time.Hour).Unix() * 1000,
        CreatedAt: time.Now().Unix() * 1000,
    }
    
    db := database.GetDB()
    err = db.Create(order).Error
    if err != nil {
        if err.Error() != "" {
            fmt.Println("订单可能已存在，或创建失败:", err)
        } else {
            fmt.Println("✅ 测试订单创建成功:", order.OrderId)
        }
    } else {
        fmt.Println("✅ 测试订单创建成功:", order.OrderId)
    }
}
```

运行：
```bash
# 在 x-ui 项目根目录
go run create_test_order.go
```

## 🧪 完整测试流程

```bash
# 终端 1：启动服务
cd backend-proxy
./backend-proxy

# 终端 2：创建测试订单
cd ../..  # 回到 x-ui 根目录
go run create_test_order.go

# 终端 2：测试接口
cd backend-proxy
curl http://localhost:8080/health
curl -X POST http://localhost:8080/api/v1/inbound/create \
  -H "Content-Type: application/json" \
  -d '{"orderId":"ORDER_TEST_001","userId":"USER_TEST_001","protocol":"vmess"}'
```

## ✅ 验证清单

- [ ] 服务编译成功
- [ ] 服务启动无错误
- [ ] API Key 自动创建成功（查看启动日志）
- [ ] 健康检查接口正常返回
- [ ] 测试订单创建成功
- [ ] 创建入站接口调用成功
- [ ] 查询订单状态接口正常

## 🔧 常见问题

### 编译失败

```bash
# 确保依赖已下载
go mod download
go mod tidy
```

### 服务无法启动

```bash
# 查看详细错误
./backend-proxy 2>&1 | tee error.log

# 检查端口是否被占用
lsof -i :8080
```

### 数据库路径错误

```bash
# 检查 x-ui 数据库是否存在
ls -la /etc/x-ui/x-ui.db

# 如果路径不同，修改 .env 文件中的 XUI_DB_PATH
```

### API Key 创建失败

- 确认 x-ui 数据库已初始化
- 确认有读取数据库文件的权限
- 检查日志中的具体错误信息

## 📚 更多信息

- 详细测试文档：`TESTING.md`
- 自动创建 API Key 说明：`README_AUTO_KEY.md`
- 完整使用指南：`README.md`

