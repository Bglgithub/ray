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

如果所有检查通过，就可以开始测试了！

