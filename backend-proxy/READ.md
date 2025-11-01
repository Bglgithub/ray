# 服务运行要求

## 必需服务

为了确保 `/api/v1/inbound/create` 接口正常工作，**必须同时运行以下两个服务**：

### 1. x-ui 服务（必需）✅

**作用**：
- 提供实际的入站配置创建功能
- 管理数据库（订单、入站配置等）
- 处理实际的业务逻辑

**为什么必需**：
- backend-proxy 只是一个**代理层**，它本身不创建入站配置
- backend-proxy 接收客户端请求后，会**调用 x-ui 的 API** 来完成实际工作
- 如果 x-ui 未运行，backend-proxy 虽然可以启动，但所有 API 调用都会失败

**启动方式**：
```bash
cd /Users/guolinbao/projects/x-ui/x-ui
./x-ui
```

**验证是否运行**：
```bash
curl http://localhost:54321/login
# 应该返回 HTML 登录页面
```

**默认端口**：`54321`

---

### 2. backend-proxy 服务（必需）✅

**作用**：
- 接收 Android 客户端的请求
- 对客户端隐藏 x-ui 的 API Key 和 Secret
- 简化客户端请求（只需要 orderId 和 userId）
- 调用 x-ui API 完成实际工作

**为什么必需**：
- 这是 Android 客户端访问的入口
- 如果 backend-proxy 未运行，客户端无法调用接口

**启动方式**：
```bash
cd /Users/guolinbao/projects/x-ui/x-ui/backend-proxy
./backend-proxy
```

**验证是否运行**：
```bash
curl http://localhost:8080/health
# 应该返回: {"success":true,"msg":"服务运行正常"}
```

**默认端口**：`8080`

---

## 服务依赖关系

```
┌─────────────┐
│ Android App │
└─────────────┘
      │
      │ HTTP 请求
      ▼
┌─────────────┐      ┌─────────────┐
│backend-proxy│─────▶│   x-ui      │
│   :8080     │ API  │   :54321    │
└─────────────┘      └─────────────┘
      │                    │
      │                    ▼
      │            ┌─────────────┐
      │            │  数据库      │
      │            │ x-ui.db     │
      │            └─────────────┘
      │
      ▼
┌─────────────┐
│返回结果给客户端│
└─────────────┘
```

**关键点**：
1. **Android App** → **backend-proxy**：客户端直接调用 backend-proxy
2. **backend-proxy** → **x-ui**：backend-proxy 内部调用 x-ui API（使用 API Key 认证）
3. **x-ui** → **数据库**：x-ui 读写数据库

**如果任何一环断开，整个流程都会失败！**

---

## 快速检查服务状态

使用提供的脚本快速检查：

```bash
cd /Users/guolinbao/projects/x-ui/x-ui/backend-proxy
./check_services.sh
```

脚本会检查：
- ✅ backend-proxy 是否运行
- ✅ x-ui 是否运行
- ✅ 数据库文件是否存在

---

## 启动顺序建议

虽然两个服务没有严格的启动顺序要求，但建议：

1. **先启动 x-ui**（确保数据库已初始化）
2. **再启动 backend-proxy**（可以自动获取 API Key）

如果先启动 backend-proxy：
- backend-proxy 会尝试自动创建 API Key
- 这需要访问 x-ui 的数据库
- 如果 x-ui 还没初始化数据库，可能会失败

---

## 常见问题

### Q1: 只启动 backend-proxy 可以吗？

**不可以**。backend-proxy 只是代理层，它需要调用 x-ui 的 API 才能完成实际工作。

### Q2: 只启动 x-ui 可以吗？

从技术上说可以，但：
- 客户端无法通过 backend-proxy 调用接口
- 客户端需要直接访问 x-ui，暴露 API Key 和 Secret（不安全）

### Q3: backend-proxy 启动失败，说无法创建 API Key

**原因**：
- x-ui 服务未启动，数据库未初始化
- 数据库路径配置错误

**解决**：
1. 先启动 x-ui 服务，确保数据库已创建
2. 检查 `.env` 文件中的 `XUI_DB_PATH` 是否正确
3. 或者手动在 `.env` 文件中设置 `XUI_API_KEY` 和 `XUI_API_SECRET`

### Q4: 接口调用返回 "x-ui API 返回错误"

**原因**：
- x-ui 服务未运行
- x-ui 地址配置错误

**检查**：
```bash
# 检查 x-ui 是否运行
curl http://localhost:54321/login

# 检查 backend-proxy 的配置
cat backend-proxy/.env | grep XUI_SERVER_URL
```

### Q5: 两个服务可以在不同机器上吗？

**可以**，但需要：
1. 确保网络连通
2. 在 `backend-proxy/.env` 中配置正确的 `XUI_SERVER_URL`
3. 确保 API Key 在 x-ui 中存在（因为 backend-proxy 需要访问数据库，如果数据库在不同机器，需要考虑如何访问）

---

## 测试前检查清单

在测试接口之前，确保：

- [ ] x-ui 服务正在运行
- [ ] backend-proxy 服务正在运行
- [ ] 两个服务都能正常访问
- [ ] 数据库文件存在
- [ ] API Key 已配置或自动创建成功

**快速检查命令**：
```bash
cd backend-proxy
./check_services.sh
```

如果所有检查通过，就可以开始测试了！

