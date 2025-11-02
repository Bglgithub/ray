# 快速部署指南（推荐方案）

## 问题说明

从 macOS 交叉编译到 Linux 比较复杂，特别是使用 CGO 时。即使有 musl-cross，也可能遇到各种兼容性问题。

## 推荐方案：使用服务器端部署脚本

最简单、最可靠的方案是使用已在服务器上的 `remote-build-deploy.sh` 脚本。

### 一键部署流程

```bash
# 1. 本地开发完成后，提交代码
git add .
git commit -m "更新功能"
git push origin main

# 2. SSH 登录服务器
ssh -i /Users/guolinbao/Desktop/vps/ula-rsa.pem -p 22 root@16.176.193.236

# 3. 执行部署（首次需要设置 Git 仓库地址）
export GIT_REPO_URL="https://github.com/your-username/x-ui.git"
~/remote-build-deploy.sh
```

### 设置 Git 仓库地址（首次使用）

在服务器上创建一个配置文件，这样以后就不需要每次都设置了：

```bash
# SSH 登录服务器后
cat > ~/.x-ui-deploy.conf << 'EOF'
export GIT_REPO_URL="https://github.com/your-username/x-ui.git"
export BRANCH="main"
EOF

# 编辑脚本，在开头加载配置
nano ~/remote-build-deploy.sh

# 在脚本开头（#!/bin/bash 之后）添加：
source ~/.x-ui-deploy.conf 2>/dev/null || true
```

然后以后直接执行：
```bash
~/remote-build-deploy.sh
```

## 服务器端脚本的优势

✅ **无需交叉编译工具**：直接在 Linux 环境编译  
✅ **避免兼容性问题**：本地编译，无需处理交叉编译的复杂性  
✅ **自动化流程**：自动拉取代码、编译、部署、重启  
✅ **自动备份**：部署前自动备份旧文件  
✅ **错误恢复**：如果新版本有问题，可以快速恢复  

## 其他方案（如果必须本地编译）

如果你想在本地编译，需要解决 musl-cross 的问题，但这比较复杂且可能不稳定。

**建议**：直接使用服务器端脚本，这是最可靠的方案。

