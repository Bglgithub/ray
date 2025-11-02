#!/bin/bash

# 上传部署脚本到服务器

# 从 .deploy.conf 读取配置
CONFIG_FILE=".deploy.conf"
if [ ! -f "$CONFIG_FILE" ]; then
    echo "错误: 配置文件 $CONFIG_FILE 不存在"
    exit 1
fi

source "$CONFIG_FILE"

# 构建 SSH 和 SCP 命令
SSH_CMD="ssh"
SCP_CMD="scp"

if [ -n "$SSH_KEY" ]; then
    SSH_CMD="$SSH_CMD -i $SSH_KEY"
    SCP_CMD="$SCP_CMD -i $SSH_KEY"
fi

if [ -n "$REMOTE_PORT" ]; then
    SSH_CMD="$SSH_CMD -p $REMOTE_PORT"
    SCP_CMD="$SCP_CMD -P $REMOTE_PORT"
fi

echo "上传部署脚本到服务器..."

# 上传脚本到 ubuntu 用户目录
$SCP_CMD remote-build-deploy.sh "${REMOTE_USER}@${REMOTE_HOST}:~/"

# 同时复制到 root 目录（如果需要用 root 执行）
$SSH_CMD "${REMOTE_USER}@${REMOTE_HOST}" "sudo cp ~/remote-build-deploy.sh /root/remote-build-deploy.sh && sudo chmod +x /root/remote-build-deploy.sh && sudo chmod +x ~/remote-build-deploy.sh" 2>/dev/null || \
$SSH_CMD "${REMOTE_USER}@${REMOTE_HOST}" "chmod +x ~/remote-build-deploy.sh"

echo "✅ 部署脚本已上传到服务器"
echo "   - ubuntu 用户目录: ~/remote-build-deploy.sh"
echo "   - root 用户目录: /root/remote-build-deploy.sh (如果可用)"
echo ""
echo "使用方法:"
echo "1. SSH 登录服务器: $SSH_CMD ${REMOTE_USER}@${REMOTE_HOST}"
echo "2. 编辑脚本设置 Git 仓库地址，或设置环境变量:"
echo "   export GIT_REPO_URL='your-git-repo-url'"
echo "   export BRANCH='main'  # 可选，默认 main"
echo "3. 执行脚本: ~/remote-build-deploy.sh"
echo ""

