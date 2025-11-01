# x-ui API 接口文档

本文档详细列出了 x-ui 工程提供的所有 API 接口。

> **注意**：所有接口路径均基于配置的 `basePath`，默认情况下 `basePath` 可能为空或 `/`。

---

## 目录

- [认证相关接口](#认证相关接口)
- [入站管理接口](#入站管理接口)
- [服务器管理接口](#服务器管理接口)
- [系统设置接口](#系统设置接口)
- [页面路由](#页面路由)
- [响应格式](#响应格式)
- [认证机制](#认证机制)
- [接口调用示例](#接口调用示例)

---

## 认证相关接口

**控制器**: `IndexController`  
**基础路径**: `/`（或配置的 basePath）

### 1. 首页

- **方法**: `GET`
- **路径**: `/`
- **说明**: 
  - 如果用户已登录，重定向到 `/xui/`
  - 如果用户未登录，显示登录页面
- **是否需要登录**: 否

### 2. 用户登录

- **方法**: `POST`
- **路径**: `/login`
- **说明**: 用户登录验证
- **是否需要登录**: 否
- **请求参数**:
  ```json
  {
    "username": "string",  // 用户名
    "password": "string"   // 密码
  }
  ```
- **响应**: 标准 JSON 格式（见下方响应格式）

### 3. 退出登录

- **方法**: `GET`
- **路径**: `/logout`
- **说明**: 清除用户会话并退出登录
- **是否需要登录**: 否
- **响应**: 重定向到登录页面

---

## 入站管理接口

**控制器**: `InboundController`  
**基础路径**: `/inbound`  
**需要登录**: ✅ 是

### 1. 获取入站列表

- **方法**: `POST`
- **路径**: `/inbound/list`
- **说明**: 获取当前用户的所有入站配置列表
- **请求参数**: 无
- **响应数据**: Inbound 对象数组

### 2. 添加入站配置

- **方法**: `POST`
- **路径**: `/inbound/add`
- **说明**: 添加新的入站配置，成功后自动重启 xray
- **请求参数** (JSON):
  ```json
  {
    "port": 443,                    // 端口号（必填，唯一）
    "protocol": "vmess",            // 协议类型：vmess/vless/trojan/shadowsocks/http/socks/Dokodemo-door
    "settings": "{}",                // 协议配置（JSON字符串）
    "streamSettings": "{}",         // 传输配置（JSON字符串）
    "sniffing": "{}",               // 流量嗅探配置（JSON字符串）
    "remark": "节点名称",           // 备注名称
    "listen": "",                   // 监听地址（可选）
    "enable": true,                 // 是否启用（默认true）
    "expiryTime": 0,                // 过期时间（时间戳，0表示永不过期）
    "total": 0,                     // 总流量限制（字节，0表示无限制）
    "up": 0,                        // 上行流量（字节）
    "down": 0                       // 下行流量（字节）
  }
  ```
- **说明**: 
  - 添加时自动生成 `tag`（格式：`inbound-{port}`）
  - 添加成功后会自动标记需要重启 xray

### 3. 删除入站配置

- **方法**: `POST`
- **路径**: `/inbound/del/:id`
- **说明**: 删除指定ID的入站配置，成功后自动重启 xray
- **路径参数**: 
  - `id` (int): 入站配置的ID
- **示例**: `/inbound/del/1`

### 4. 更新入站配置

- **方法**: `POST`
- **路径**: `/inbound/update/:id`
- **说明**: 更新指定ID的入站配置，成功后自动重启 xray
- **路径参数**: 
  - `id` (int): 入站配置的ID
- **请求参数**: 同"添加入站配置"，需要包含 `id` 字段
- **示例**: `/inbound/update/1`

---

## 服务器管理接口

**控制器**: `ServerController`  
**基础路径**: `/server`  
**需要登录**: ✅ 是

### 1. 获取服务器状态

- **方法**: `POST`
- **路径**: `/server/status`
- **说明**: 获取服务器运行状态（CPU、内存、网络、xray状态等）
- **请求参数**: 无
- **响应数据**: Status 对象
  ```json
  {
    "cpu": 0.0,           // CPU使用率
    "mem": 0.0,           // 内存使用率
    "disk": 0.0,          // 磁盘使用率
    "uptime": 0,          // 系统运行时间
    "load": "",           // 系统负载
    "xray": {             // xray状态
      "status": "",       // 运行状态
      "version": "",      // 版本号
      ...
    }
  }
  ```
- **说明**: 
  - 状态信息每2秒自动刷新一次
  - 如果超过3分钟没有请求，会停止自动刷新

### 2. 获取 Xray 版本列表

- **方法**: `POST`
- **路径**: `/server/getXrayVersion`
- **说明**: 获取可用的 xray 版本列表
- **请求参数**: 无
- **响应数据**: 版本号字符串数组，例如：`["v1.4.2", "v1.4.1", ...]`
- **说明**: 结果会缓存1分钟，避免频繁请求

### 3. 安装 Xray

- **方法**: `POST`
- **路径**: `/server/installXray/:version`
- **说明**: 安装指定版本的 xray
- **路径参数**: 
  - `version` (string): xray 版本号，例如：`v1.4.2`
- **示例**: `/server/installXray/v1.4.2`

---

## 系统设置接口

**控制器**: `SettingController`  
**基础路径**: `/setting`  
**需要登录**: ✅ 是

### 1. 获取所有设置

- **方法**: `POST`
- **路径**: `/setting/all`
- **说明**: 获取所有系统设置项
- **请求参数**: 无
- **响应数据**: AllSetting 对象（包含所有系统配置）

### 2. 更新系统设置

- **方法**: `POST`
- **路径**: `/setting/update`
- **说明**: 更新系统设置
- **请求参数**: AllSetting 对象（JSON）
  ```json
  {
    "webPort": 54321,              // Web面板端口
    "webListen": "",               // Web面板监听地址
    "webBasePath": "",             // Web面板基础路径
    "webCertFile": "",             // SSL证书文件路径
    "webKeyFile": "",              // SSL私钥文件路径
    "xrayTemplateConfig": "",      // Xray配置模板
    ...
  }
  ```

### 3. 更新登录用户

- **方法**: `POST`
- **路径**: `/setting/updateUser`
- **说明**: 更新登录用户名和密码
- **请求参数**:
  ```json
  {
    "oldUsername": "admin",        // 原用户名（必填）
    "oldPassword": "admin",        // 原密码（必填）
    "newUsername": "newadmin",     // 新用户名（必填）
    "newPassword": "newpassword"   // 新密码（必填）
  }
  ```
- **说明**: 
  - 需要验证原用户名和密码
  - 新用户名和新密码不能为空

### 4. 重启面板

- **方法**: `POST`
- **路径**: `/setting/restartPanel`
- **说明**: 重启 x-ui 面板服务
- **请求参数**: 无
- **说明**: 面板会在3秒后重启

---

## 页面路由

**控制器**: `XUIController`  
**基础路径**: `/xui`  
**需要登录**: ✅ 是

> **注意**：这些是 HTML 页面路由，不是 API 接口，返回的是 HTML 页面。

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/xui/` | 系统状态页面 |
| GET | `/xui/inbounds` | 入站列表管理页面 |
| GET | `/xui/setting` | 系统设置页面 |

---

## 响应格式

所有 POST API 接口返回标准 JSON 格式：

### 成功响应

```json
{
  "success": true,
  "msg": "操作成功",
  "obj": {
    // 对象数据（如果有）
  }
}
```

### 失败响应

```json
{
  "success": false,
  "msg": "操作失败: 错误信息",
  "obj": null
}
```

### 字段说明

- `success` (boolean): 操作是否成功
- `msg` (string): 操作结果消息
- `obj` (object): 返回的数据对象，可能为 `null` 或具体数据

---

## 认证机制

### Session 管理

- 所有需要登录的接口都使用 `checkLogin` 中间件
- 登录状态通过 session cookie 管理
- Session 存储位置：cookie（使用 Gin 的 sessions 中间件）

### 未登录时的行为

- **AJAX 请求**: 返回 JSON 错误消息 `"登录时效已过，请重新登录"`
- **普通请求**: 重定向到登录页面

### 登录验证流程

1. 客户端发送 POST `/login` 请求，包含用户名和密码
2. 服务端验证用户凭据
3. 验证成功后，设置 session cookie
4. 后续请求携带 session cookie 即可访问需要登录的接口

---

## 接口调用示例

### 1. 用户登录

```bash
curl -X POST http://localhost:54321/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "admin"
  }'

# 响应示例
{
  "success": true,
  "msg": "登录成功",
  "obj": null
}
```

### 2. 获取入站列表

```bash
curl -X POST http://localhost:54321/inbound/list \
  -H "Content-Type: application/json" \
  -b "session=<session-cookie>"

# 响应示例
{
  "success": true,
  "msg": "获取成功",
  "obj": [
    {
      "id": 1,
      "port": 443,
      "protocol": "vmess",
      "remark": "节点1",
      "enable": true,
      ...
    }
  ]
}
```

### 3. 添加入站配置

```bash
curl -X POST http://localhost:54321/inbound/add \
  -H "Content-Type: application/json" \
  -b "session=<session-cookie>" \
  -d '{
    "port": 8443,
    "protocol": "vmess",
    "settings": "{\"clients\":[...],\"disableInsecureEncryption\":false}",
    "streamSettings": "{\"network\":\"ws\"}",
    "remark": "测试节点",
    "enable": true
  }'
```

### 4. 获取服务器状态

```bash
curl -X POST http://localhost:54321/server/status \
  -H "Content-Type: application/json" \
  -b "session=<session-cookie>"

# 响应示例
{
  "success": true,
  "msg": "",
  "obj": {
    "cpu": 12.5,
    "mem": 45.2,
    "disk": 30.0,
    "uptime": 86400,
    "xray": {
      "status": "running",
      "version": "1.4.2"
    }
  }
}
```

### 5. 更新用户密码

```bash
curl -X POST http://localhost:54321/setting/updateUser \
  -H "Content-Type: application/json" \
  -b "session=<session-cookie>" \
  -d '{
    "oldUsername": "admin",
    "oldPassword": "admin",
    "newUsername": "admin",
    "newPassword": "newpassword123"
  }'
```

---

## 注意事项

1. **路径前缀**: 所有接口路径都基于配置的 `basePath`，可能需要根据实际配置调整路径
2. **自动重启**: 添加、删除、更新入站配置后，系统会自动标记需要重启 xray，并在10秒内自动重启
3. **会话超时**: 如果登录会话过期，需要重新登录获取新的 session cookie
4. **HTTPS 支持**: 如果配置了 SSL 证书，所有接口都应该通过 HTTPS 访问
5. **端口冲突**: 添加入站配置时，端口号必须唯一，不能与其他入站配置重复

---

## 支持的协议

x-ui 支持以下协议类型：

- `vmess` - VMess 协议
- `vless` - VLESS 协议
- `trojan` - Trojan 协议
- `shadowsocks` - Shadowsocks 协议
- `http` - HTTP 代理
- `socks` - SOCKS 代理
- `Dokodemo-door` - 任意门协议

---

## 版本信息

- 文档创建时间: 2024
- 基于代码版本: 查看 `config/version` 文件获取具体版本号

---

## 相关文件

- 控制器代码: `web/controller/`
- 服务层代码: `web/service/`
- 数据模型: `database/model/model.go`
- 路由配置: `web/web.go`

---

**最后更新**: 请根据实际代码更新此文档

