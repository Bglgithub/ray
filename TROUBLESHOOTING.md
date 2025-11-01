# x-ui 服务启动失败排查指南

## 当前状态分析

你的服务状态显示：
```
Active: activating (auto-restart) (Result: exit-code)
status=1/FAILURE
```

这表明服务启动时遇到了错误并立即退出，systemd 正在自动重启。

## 排查步骤

### 步骤 1：查看详细错误日志

在服务器上执行以下命令查看具体错误：

```bash
# 查看最近的错误日志（最重要！）
journalctl -u x-ui -n 100 --no-pager

# 或者查看实时日志
journalctl -u x-ui -f
```

这会显示服务启动时的具体错误信息。

### 步骤 2：手动运行测试

停止服务，手动运行查看输出：

```bash
# 停止服务
systemctl stop x-ui

# 手动运行（会显示详细错误信息）
cd /usr/local/x-ui
./x-ui
```

### 步骤 3：常见问题和解决方案

根据错误日志的不同，可能的原因和解决方案：

#### 问题 1：缺少依赖文件

**错误信息**：`file not found` 或 `no such file`

**检查**：
```bash
# 检查 xray 二进制文件是否存在
ls -lh /usr/local/x-ui/xray-linux-*

# x-ui 需要这些文件，如果没有需要从项目中复制
# 或者让 x-ui 自动下载（如果有这个功能）
```

#### 问题 2：数据库路径错误或权限问题

**错误信息**：`cannot open database` 或 `permission denied`

**检查**：
```bash
# 检查数据库目录
ls -ld /etc/x-ui
ls -lh /etc/x-ui/x-ui.db 2>/dev/null || echo "数据库文件不存在"

# 创建目录（如果不存在）
sudo mkdir -p /etc/x-ui
sudo chown root:root /etc/x-ui
```

#### 问题 3：端口被占用

**错误信息**：`bind: address already in use` 或 `port already in use`

**检查**：
```bash
# 检查端口占用
ss -tulpn | grep 54321
netstat -tulpn | grep 54321

# 如果被占用，停止占用端口的程序或修改 x-ui 配置
```

#### 问题 4：配置文件格式错误

**错误信息**：`config error` 或 `parse error`

**检查**：
```bash
# 检查配置文件（如果有）
cat /usr/local/x-ui/config.json
```

#### 问题 5：工作目录问题

**错误**：systemd 服务的工作目录配置可能不正确

**检查服务文件**：
```bash
cat /etc/systemd/system/x-ui.service
```

**修复**（如果需要）：
```bash
sudo nano /etc/systemd/system/x-ui.service
```

确保包含：
```ini
[Service]
WorkingDirectory=/usr/local/x-ui
```

然后：
```bash
sudo systemctl daemon-reload
sudo systemctl restart x-ui
```

#### 问题 6：缺少系统库

**错误信息**：`cannot find shared library` 或 `undefined symbol`

**检查**：
```bash
# 检查文件类型
file /usr/local/x-ui/x-ui

# 检查依赖库（Linux）
ldd /usr/local/x-ui/x-ui 2>/dev/null || echo "不是动态链接"
```

如果是静态编译的（应该包含 `statically linked`），则不需要系统库。

#### 问题 7：内存不足

**错误信息**：`out of memory` 或 `cannot allocate memory`

**检查**：
```bash
free -h
```

## 快速修复尝试

### 尝试 1：检查文件权限

```bash
# 确保文件有执行权限
sudo chmod +x /usr/local/x-ui/x-ui

# 确保文件所有者正确
sudo chown root:root /usr/local/x-ui/x-ui
```

### 尝试 2：检查依赖文件

x-ui 可能需要 xray 二进制文件。检查是否有这些文件：

```bash
ls -lh /usr/local/x-ui/bin/
```

如果没有，需要从源码目录复制：
```bash
# 在本地项目目录
scp -i [密钥] bin/xray-linux-amd64 root@服务器:/usr/local/x-ui/bin/

# 或者在服务器上让 x-ui 自动下载
```

### 尝试 3：创建必要的目录和文件

```bash
# 创建配置目录
sudo mkdir -p /etc/x-ui
sudo mkdir -p /usr/local/x-ui/bin

# 确保有写入权限
sudo chmod 755 /etc/x-ui
sudo chmod 755 /usr/local/x-ui
```

### 尝试 4：修改服务配置

编辑服务文件，添加更多调试信息：

```bash
sudo nano /etc/systemd/system/x-ui.service
```

修改为：
```ini
[Unit]
Description=x-ui Service
After=network.target

[Service]
Type=simple
WorkingDirectory=/usr/local/x-ui
ExecStart=/usr/local/x-ui/x-ui
Restart=always
RestartSec=5s
User=root
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
```

然后：
```bash
sudo systemctl daemon-reload
sudo systemctl restart x-ui
journalctl -u x-ui -f  # 查看实时日志
```

## 需要你提供的信息

请执行以下命令并告诉我结果：

```bash
# 1. 错误日志
journalctl -u x-ui -n 50 --no-pager

# 2. 手动运行输出
cd /usr/local/x-ui
./x-ui

# 3. 文件信息
ls -lh /usr/local/x-ui/x-ui
file /usr/local/x-ui/x-ui

# 4. 目录结构
ls -la /usr/local/x-ui/
```

把这些信息发给我，我可以更准确地帮你定位问题。

