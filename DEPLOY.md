# 一键部署脚本使用指南

## 功能说明

`deploy.sh` 脚本可以：
1. 自动编译 x-ui 和 backend-proxy 服务
2. 上传编译好的二进制文件到远程服务器
3. 自动创建 systemd 服务（可选）
4. 设置文件权限

## 快速开始

### 1. 配置部署信息

```bash
# 复制配置文件示例
cp .deploy.conf.example .deploy.conf

# 编辑配置文件
nano .deploy.conf
```

### 2. 配置文件说明

```bash
# 远程服务器信息（必填）
REMOTE_HOST="192.168.1.100"      # 服务器IP或域名
REMOTE_USER="root"               # SSH用户名
REMOTE_PORT="22"                 # SSH端口（默认22）

# 部署路径
XUI_DEPLOY_PATH="/usr/local/x-ui"
BACKEND_PROXY_DEPLOY_PATH="/usr/local/backend-proxy"

# 编译选项
BUILD_ARCH="amd64"               # CPU架构: amd64, arm64, 386
BUILD_OS="linux"                 # 操作系统: linux, darwin, windows

# SSH 密钥（可选，如果使用密码登录则留空）
SSH_KEY="~/.ssh/id_rsa"          # SSH密钥路径

# 是否创建 systemd 服务
CREATE_SYSTEMD_SERVICE="yes"     # yes 或 no
```

### 3. 运行部署脚本

```bash
./deploy.sh
```

## 部署流程

脚本会自动执行以下步骤：

1. ✅ **编译 x-ui**：在项目根目录编译 x-ui 服务
2. ✅ **编译 backend-proxy**：编译 backend-proxy 服务
3. ✅ **测试连接**：验证 SSH 连接是否正常
4. ✅ **创建目录**：在远程服务器创建部署目录
5. ✅ **上传文件**：上传编译好的二进制文件
6. ✅ **创建服务**：自动创建 systemd 服务（如果启用）

## 部署后配置

### 步骤 1：SSH 登录服务器

```bash
# 使用配置文件中的信息登录
ssh -i [SSH_KEY路径] -p [SSH端口] [用户名]@[服务器地址]

# 示例（如果使用密钥）
ssh -i ~/.ssh/id_rsa -p 22 root@16.176.193.236

# 示例（如果使用密码）
ssh -p 22 root@16.176.193.236
```

### 步骤 2：配置 backend-proxy 环境变量

登录服务器后：

```bash
# 进入 backend-proxy 目录
cd /usr/local/x-ui/backend-proxy

# 创建配置文件
cp env.example .env

# 编辑配置
nano .env
```

**关键配置项**（必须配置）：
```env
# x-ui 服务器地址
XUI_SERVER_URL=http://localhost:54321

# x-ui 数据库路径（默认路径）
XUI_DB_PATH=/etc/x-ui/x-ui.db

# 后端代理服务端口
PROXY_PORT=8080

# 默认协议（vmess/vless/trojan/shadowsocks）
DEFAULT_PROTOCOL=shadowsocks

# API Key 和 Secret（如果为空，服务启动时会自动创建）
XUI_API_KEY=
XUI_API_SECRET=
```

**保存配置文件**：
- `nano` 编辑器：按 `Ctrl+O` 保存，`Ctrl+X` 退出
- `vim` 编辑器：按 `i` 进入编辑模式，`Esc` 退出编辑，`:wq` 保存并退出

### 步骤 3：启动服务

**如果使用 systemd（推荐）**：
```bash
# 启动服务
systemctl start x-ui
systemctl start backend-proxy

# 查看状态
systemctl status x-ui
systemctl status backend-proxy

# 设置开机自启
systemctl enable x-ui
systemctl enable backend-proxy

# 查看日志
journalctl -u x-ui -f
journalctl -u backend-proxy -f
```

**如果手动启动**：
```bash
# 启动 x-ui
cd /usr/local/x-ui
./x-ui &

# 启动 backend-proxy
cd /usr/local/backend-proxy
./backend-proxy &
```

### 步骤 4：验证服务

**验证服务是否正常启动**：

```bash
# 检查 x-ui 服务状态（默认端口 54321）
systemctl status x-ui

# 检查 backend-proxy 服务状态（默认端口 8080）
systemctl status backend-proxy

# 或者使用 curl 测试（如果服务已启动）
curl http://localhost:54321/login
curl http://localhost:8080/health
```

**预期响应**：
- x-ui：返回登录页面的 HTML
- backend-proxy：返回 `{"success":true,"msg":"服务运行正常"}`

### 步骤 5：设置开机自启（可选但推荐）

```bash
# 设置 x-ui 开机自启
systemctl enable x-ui

# 设置 backend-proxy 开机自启
systemctl enable backend-proxy

# 验证开机自启设置
systemctl is-enabled x-ui
systemctl is-enabled backend-proxy
```

### 步骤 6：查看服务日志（如遇问题）

```bash
# 查看 x-ui 实时日志
journalctl -u x-ui -f

# 查看 backend-proxy 实时日志
journalctl -u backend-proxy -f

# 查看最近的 50 行日志
journalctl -u x-ui -n 50
journalctl -u backend-proxy -n 50
```

## 完整部署检查清单

部署完成后，请确认以下所有步骤：

- [ ] ✅ 服务文件已上传到服务器
- [ ] ✅ systemd 服务已创建（如果启用）
- [ ] ✅ backend-proxy 配置文件 `.env` 已创建并配置
- [ ] ✅ x-ui 服务已启动
- [ ] ✅ backend-proxy 服务已启动
- [ ] ✅ 服务状态正常（`systemctl status`）
- [ ] ✅ 端口可以访问（curl 测试）
- [ ] ✅ 开机自启已设置（可选）

## 服务启动问题排查

### ⚠️ 重要说明

**x-ui 和 backend-proxy 不是 Docker 容器**，它们是直接运行在系统上的二进制服务，通过 systemd 管理。

**不要使用 `docker ps` 检查**，应该使用 `systemctl` 命令。

### 检查服务状态

```bash
# 检查服务是否运行
systemctl status x-ui
systemctl status backend-proxy

# 查看所有相关服务
systemctl list-units | grep -E "(x-ui|backend-proxy)"

# 检查进程
ps aux | grep -E "(x-ui|backend-proxy)" | grep -v grep
```

### 如果服务未启动，排查步骤

**1. 检查服务文件是否存在**

```bash
ls -l /etc/systemd/system/x-ui.service
ls -l /etc/systemd/system/backend-proxy.service
```

**2. 查看服务错误日志**

```bash
# 查看 x-ui 服务日志
journalctl -u x-ui -n 50 --no-pager

# 查看 backend-proxy 服务日志
journalctl -u backend-proxy -n 50 --no-pager
```

**3. 检查可执行文件是否存在和权限**

```bash
# 检查文件是否存在
ls -lh /usr/local/x-ui/x-ui
ls -lh /usr/local/x-ui/backend-proxy/backend-proxy

# 检查是否有执行权限
file /usr/local/x-ui/x-ui
file /usr/local/x-ui/backend-proxy/backend-proxy
```

**4. 手动测试启动（定位问题）**

```bash
# 手动运行 x-ui（前台运行，查看输出）
cd /usr/local/x-ui
./x-ui

# 手动运行 backend-proxy（前台运行，查看输出）
cd /usr/local/x-ui/backend-proxy
./backend-proxy
```

**5. 检查配置文件**

```bash
# 检查 backend-proxy 配置文件
cat /usr/local/x-ui/backend-proxy/.env

# 检查数据库文件是否存在（x-ui）
ls -lh /etc/x-ui/x-ui.db
```

**6. 检查端口占用**

```bash
# 检查 54321 端口（x-ui）
netstat -tulpn | grep 54321
# 或
ss -tulpn | grep 54321

# 检查 8080 端口（backend-proxy）
netstat -tulpn | grep 8080
# 或
ss -tulpn | grep 8080
```

### 常见启动失败原因

1. **配置文件错误**
   - backend-proxy 的 `.env` 文件配置错误
   - 数据库路径不正确

2. **权限问题**
   - 可执行文件没有执行权限
   - 无法访问数据库文件

3. **端口被占用**
   - 端口 54321 或 8080 已被其他程序占用

4. **依赖问题**
   - x-ui 需要 xray 二进制文件

5. **服务文件错误**
   - systemd 服务文件配置错误

### 重新启动服务

```bash
# 停止服务
systemctl stop x-ui
systemctl stop backend-proxy

# 重新加载配置
systemctl daemon-reload

# 启动服务
systemctl start x-ui
systemctl start backend-proxy

# 查看状态
systemctl status x-ui
systemctl status backend-proxy
```

## 常见问题

### 1. SSH 连接失败

**问题**：`无法连接到远程服务器`

**解决方案**：
- 检查服务器 IP 地址是否正确
- 确认 SSH 端口是否开放
- 检查防火墙设置
- 如果使用密钥，确认密钥路径正确

### 2. 权限不足

**问题**：创建 systemd 服务失败

**解决方案**：
- 确认使用 root 用户或具有 sudo 权限的用户
- 或者设置 `CREATE_SYSTEMD_SERVICE="no"` 手动创建服务

### 3. 编译失败

**问题**：Go 编译错误

**解决方案**：
- 确认 Go 环境已安装：`go version`
- 确认项目依赖完整：`go mod tidy`
- 检查交叉编译环境变量设置

### 4. 服务启动失败

**问题**：服务无法启动

**解决方案**：
```bash
# 查看服务日志
journalctl -u x-ui -n 50
journalctl -u backend-proxy -n 50

# 检查配置文件
cat /usr/local/backend-proxy/.env

# 检查端口占用
netstat -tulpn | grep 54321
netstat -tulpn | grep 8080
```

## 更新部署

如果需要更新服务：

```bash
# 重新运行部署脚本
./deploy.sh

# 重启服务
systemctl restart x-ui
systemctl restart backend-proxy
```

## 多服务器部署

如果需要部署到多个服务器，可以：

1. **方式一**：创建多个配置文件
   ```bash
   cp .deploy.conf .deploy.conf.server1
   cp .deploy.conf .deploy.conf.server2
   # 修改配置后分别部署
   ```

2. **方式二**：修改脚本支持配置文件参数
   ```bash
   # 在脚本中添加参数解析
   CONFIG_FILE="${1:-.deploy.conf}"
   ```

## 安全建议

1. **使用 SSH 密钥认证**：避免密码泄露
2. **限制 SSH 端口**：使用非标准端口
3. **防火墙配置**：只开放必要端口
4. **服务用户**：不使用 root 运行服务（生产环境）
5. **定期更新**：及时更新服务版本

## 手动部署（不使用脚本）

如果不想使用自动部署脚本，可以手动执行：

```bash
# 1. 本地编译
go build -o x-ui ./
cd backend-proxy && go build -o backend-proxy ./

# 2. 上传文件
scp x-ui user@server:/usr/local/x-ui/
scp backend-proxy/backend-proxy user@server:/usr/local/backend-proxy/

# 3. SSH 登录服务器
ssh user@server

# 4. 设置权限
chmod +x /usr/local/x-ui/x-ui
chmod +x /usr/local/backend-proxy/backend-proxy

# 5. 配置和启动（参考上面的步骤）
```

---

**提示**：首次部署建议先在测试环境验证，确认无误后再部署到生产环境。

