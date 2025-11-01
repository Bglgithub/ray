# 服务器端自动部署脚本使用指南

## 功能说明

`remote-build-deploy.sh` 是一个在服务器上自动编译和部署的脚本，功能包括：

1. ✅ 自动检查并安装 Go 环境
2. ✅ 从 Git 仓库拉取最新代码
3. ✅ 自动编译 x-ui 和 backend-proxy
4. ✅ 自动备份旧文件
5. ✅ 部署新编译的文件
6. ✅ 自动重启服务
7. ✅ 验证服务状态

## 快速开始

### 步骤 1：上传脚本到服务器

```bash
# 在项目根目录执行
./upload-deploy-script.sh
```

或者手动上传：

```bash
# 从 .deploy.conf 读取配置并上传
scp -i [密钥] -P [端口] remote-build-deploy.sh [用户]@16.176.193.236:~/
```

### 步骤 2：SSH 登录服务器并配置

```bash
ssh -i [密钥] -p 22 root@16.176.193.236
```

### 步骤 3：设置 Git 仓库地址

有两种方式：

#### 方式 1：编辑脚本（一次性）

```bash
nano ~/remote-build-deploy.sh
```

找到这一行并修改：
```bash
GIT_REPO_URL="${GIT_REPO_URL:-}"  # 改为你的 Git 仓库地址
```

#### 方式 2：使用环境变量（推荐）

```bash
# 设置环境变量
export GIT_REPO_URL="https://github.com/your-username/x-ui.git"
export BRANCH="main"  # 可选，默认 main

# 执行脚本
~/remote-build-deploy.sh
```

### 步骤 4：执行部署

```bash
~/remote-build-deploy.sh
```

## 完整使用示例

```bash
# 1. 上传脚本（在本地）
./upload-deploy-script.sh

# 2. SSH 登录服务器
ssh -i ~/.ssh/id_rsa root@16.176.193.236

# 3. 设置 Git 仓库地址
export GIT_REPO_URL="https://github.com/your-username/x-ui.git"
export BRANCH="main"

# 4. 执行部署
~/remote-build-deploy.sh
```

## 脚本配置说明

### 可配置的环境变量

```bash
# Git 仓库地址（必填）
export GIT_REPO_URL="https://github.com/your-username/x-ui.git"

# Git 分支（可选，默认 main）
export BRANCH="main"

# 构建目录（可选，默认 /tmp/x-ui-build）
export BUILD_DIR="/tmp/x-ui-build"

# 部署路径（可选，默认配置）
export XUI_DEPLOY_PATH="/usr/local/x-ui"
export BACKEND_PROXY_DEPLOY_PATH="/usr/local/x-ui/backend-proxy"
```

### 脚本流程

1. **检查 Go 环境**：如果未安装，自动安装 Go 1.21.6
2. **获取代码**：从 Git 仓库克隆或更新代码
3. **编译服务**：编译 x-ui 和 backend-proxy
4. **备份文件**：备份旧的二进制文件
5. **部署文件**：复制新文件到部署目录
6. **重启服务**：停止旧服务，启动新服务
7. **验证状态**：检查服务是否正常启动

## 使用场景

### 场景 1：代码更新后自动部署

```bash
# 在本地开发完成后
git push origin main

# 在服务器上执行
export GIT_REPO_URL="https://github.com/your-username/x-ui.git"
~/remote-build-deploy.sh
```

### 场景 2：创建定时自动部署

```bash
# 编辑 crontab
crontab -e

# 添加定时任务（每天凌晨 2 点自动部署）
0 2 * * * export GIT_REPO_URL="https://github.com/your-username/x-ui.git" && /root/remote-build-deploy.sh >> /var/log/x-ui-deploy.log 2>&1
```

### 场景 3：手动触发部署

```bash
# 直接执行
~/remote-build-deploy.sh
```

## 安全建议

1. **使用 SSH 密钥**：避免在脚本中硬编码密码
2. **Git 仓库权限**：使用私有仓库或设置访问令牌
3. **备份策略**：脚本会自动备份，但建议定期手动备份重要数据
4. **日志监控**：关注部署日志，及时发现问题

## 故障恢复

如果部署后服务出现问题：

```bash
# 1. 停止服务
sudo systemctl stop x-ui
sudo systemctl stop backend-proxy

# 2. 从备份恢复
sudo cp /tmp/x-ui-backup-YYYYMMDD-HHMMSS/x-ui /usr/local/x-ui/x-ui
sudo cp /tmp/x-ui-backup-YYYYMMDD-HHMMSS/backend-proxy /usr/local/x-ui/backend-proxy/backend-proxy

# 3. 重启服务
sudo systemctl start x-ui
sudo systemctl start backend-proxy
```

## 常见问题

### 1. Go 安装失败

**问题**：网络问题导致 Go 下载失败

**解决**：
```bash
# 手动下载并安装 Go
cd /tmp
wget https://go.dev/dl/go1.21.6.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.6.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
```

### 2. Git 克隆失败

**问题**：Git 仓库地址错误或需要认证

**解决**：
- 检查 Git 仓库地址是否正确
- 如果使用私有仓库，需要配置 SSH 密钥或使用访问令牌
- 或者手动克隆后再执行脚本

### 3. 编译失败

**问题**：代码有错误或依赖问题

**解决**：
```bash
# 查看编译错误
cd /tmp/x-ui-build
go build -v ./  # 查看详细编译信息
```

### 4. 服务启动失败

**问题**：新编译的文件有问题

**解决**：
- 从备份恢复旧文件
- 检查日志：`journalctl -u x-ui -n 50`
- 手动测试：`/usr/local/x-ui/x-ui`

## 高级用法

### 使用 Git Webhook 自动触发

在 Git 仓库（如 GitHub）设置 Webhook，当代码 push 时自动触发服务器上的脚本。

1. **在服务器上创建 Webhook 接收脚本**（需要 web 服务器）
2. **GitHub/GitLab 设置 Webhook URL**
3. **Webhook 调用部署脚本**

### 使用 CI/CD 集成

将脚本集成到 CI/CD 流程中：

```yaml
# .github/workflows/deploy.yml
name: Deploy
on:
  push:
    branches: [main]
jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - name: Deploy to server
        run: |
          ssh root@16.176.193.236 'export GIT_REPO_URL="${{ secrets.GIT_REPO_URL }}" && ~/remote-build-deploy.sh'
```

---

**提示**：首次使用建议先在测试环境验证脚本功能。

