package main

import (
	"bufio"
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// 配置结构
type Config struct {
	XUIServerURL    string // x-ui服务器地址，例如: http://localhost:54321
	APIKey          string // x-ui的API Key
	APISecret       string // x-ui的API Secret
	ProxyPort       string // 后端代理服务端口
	XUIDBPath       string // x-ui数据库路径
	DefaultProtocol string // 默认协议（vmess/vless/trojan/shadowsocks）
	ServerCDNAddr   string // 当前服务器CDN地址
	ServerCountry   string // 当前服务器国家代码
}

var config Config

// APIKey 模型（与x-ui中的model.APIKey保持一致）
type APIKey struct {
	Id         int    `gorm:"primaryKey;autoIncrement"`
	Key        string `gorm:"uniqueIndex;not null"`
	Secret     string `gorm:"not null"`
	Name       string
	Status     string `gorm:"default:active"`
	CreatedAt  int64  `gorm:"autoCreateTime"`
	LastUsedAt int64
	RateLimit  int `gorm:"default:100"`
	AllowedIPs string
}

// 客户端请求结构（简化版，只需要订单ID和用户ID）
type CreateInboundRequest struct {
	OrderId string `json:"orderId" binding:"required"`
	UserId  string `json:"userId" binding:"required"`
}

// x-ui API 请求结构（完整版，包含所有参数）
type XUICreateInboundRequest struct {
	OrderId        string `json:"orderId"`        // 订单号
	UserId         string `json:"userId"`         // 用户ID
	Protocol       string `json:"protocol"`       // 协议
	Port           int    `json:"port"`           // 端口（自动分配）
	Remark         string `json:"remark"`         // 备注
	ExpiryTime     int64  `json:"expiryTime"`     // 过期时间
	Total          int64  `json:"total"`          // 总流量
	Settings       string `json:"settings"`       // 设置
	StreamSettings string `json:"streamSettings"` // 流设置
	Sniffing       string `json:"sniffing"`       // 嗅探
	Listen         string `json:"listen"`         // 监听地址
}

type OrderStatusRequest struct {
	OrderId string `json:"orderId" binding:"required"`
}

// 响应结构
type Response struct {
	Success bool        `json:"success"`
	Msg     string      `json:"msg"`
	Data    interface{} `json:"data,omitempty"`
}

func init() {
	// 初始化随机数种子
	rand.Seed(time.Now().UnixNano())
}

func main() {
	// 加载环境变量
	err := godotenv.Load()
	if err != nil {
		log.Println("未找到.env文件，使用环境变量")
	}

	log.Println("[backend-proxy] 启动...")
	// 读取配置
	config.XUIServerURL = getEnv("XUI_SERVER_URL", "http://localhost:54321")
	config.APIKey = getEnv("XUI_API_KEY", "")
	config.APISecret = getEnv("XUI_API_SECRET", "")
	config.ProxyPort = getEnv("PROXY_PORT", "8080")
	config.XUIDBPath = getEnv("XUI_DB_PATH", "/etc/x-ui/x-ui.db")
	config.DefaultProtocol = getEnv("DEFAULT_PROTOCOL", "shadowsocks")
	config.ServerCDNAddr = getEnv("SERVER_CDN_ADDRESS", "")
	config.ServerCountry = getEnv("SERVER_COUNTRY_CODE", "CN")

	log.Println("[backend-proxy] 检测代理APIKey&APISecret...")
	// 如果API Key或Secret为空，尝试自动创建
	if config.APIKey == "" || config.APISecret == "" {
		log.Println("检测到API Key或Secret为空，尝试自动创建...")
		apiKey, apiSecret, err := createOrGetAPIKey()
		if err != nil {
			log.Fatalf("无法自动创建API Key: %v\n请手动设置 XUI_API_KEY 和 XUI_API_SECRET 环境变量", err)
		}
		config.APIKey = apiKey
		config.APISecret = apiSecret
		log.Printf("✅ 自动创建API Key成功: %s", apiKey)
	}

	// 初始化服务器站点数据库表
	log.Println("[backend-proxy] 初始化服务器站点数据库表...")
	if err := initServerSiteDB(); err != nil {
		log.Printf("警告: 初始化服务器站点数据库表失败: %v", err)
	} else {
		// 自动插入当前服务器到数据库
		if err := autoInsertCurrentServer(); err != nil {
			log.Printf("警告: 自动插入当前服务器失败: %v", err)
		}
	}

	// 设置Gin模式
	gin.SetMode(gin.ReleaseMode)

	// 创建路由
	r := gin.Default()

	// 添加CORS支持（如果需要）
	r.Use(corsMiddleware())

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, Response{
			Success: true,
			Msg:     "服务运行正常",
		})
	})

	// 测试接口：快速创建/更新订单（方便测试，无需认证）
	r.GET("/test/create-order", createOrUpdateOrderTest)

	// 测试接口：快速创建服务器站点（方便测试，无需认证）
	r.GET("/test/create-server", createServerSiteTest)

	// API路由
	api := r.Group("/api/v1")
	{
		api.POST("/inbound/create", createInboundProxy)
		api.POST("/order/status", getOrderStatusProxy)

		// 服务器站点管理接口
		api.GET("/servers", getServerSitesList) // 获取服务器列表
	}

	// 启动服务
	log.Printf("后端代理服务启动在端口 %s", config.ProxyPort)
	log.Printf("x-ui服务器地址: %s", config.XUIServerURL)
	log.Printf("APIKey: %s", config.APIKey)
	log.Printf("APISecret: %s", config.APISecret)
	log.Println("[backend-proxy] 启动成功")
	log.Fatal(r.Run(":" + config.ProxyPort))
}

// 创建入站配置代理
func createInboundProxy(c *gin.Context) {
	var req CreateInboundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Msg:     "请求参数错误: " + err.Error(),
		})
		return
	}

	// 构建完整的 x-ui API 请求（填充默认值）
	xuiReq, err := buildXUIRequest(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Msg:     "构建请求失败: " + err.Error(),
		})
		return
	}

	// 调用x-ui API
	response, err := callXUIAPI("/api/v1/inbound/create", xuiReq)
	if err != nil {
		log.Printf("调用x-ui API失败: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Msg:     "创建入站配置失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// 构建 x-ui API 请求，填充默认值
func buildXUIRequest(req CreateInboundRequest) (*XUICreateInboundRequest, error) {
	// 先从订单中获取信息（如果订单有配置）
	order, err := getOrderInfo(req.OrderId)
	if err != nil {
		log.Printf("获取订单信息失败（将使用默认值）: %v", err)
		// 即使获取订单失败，也继续创建，使用默认值
		return nil, fmt.Errorf("不存在该订单: %v", req.OrderId)

	}

	if order.UserId != req.UserId {
		return nil, fmt.Errorf("订单用户ID与请求用户ID不匹配: %s != %s", order.UserId, req.UserId)
	}

	xuiReq := &XUICreateInboundRequest{
		OrderId:        req.OrderId,
		UserId:         req.UserId,
		Protocol:       config.DefaultProtocol, // 使用配置的默认协议
		Port:           0,                      // 0 表示自动分配端口
		Remark:         generateRemark(req.UserId, req.OrderId),
		ExpiryTime:     order.ExpiresAt, // 从订单中获取，如果没有则为0（永不过期）
		Total:          0,               // 从订单中获取，如果没有则为0（无限制）
		Settings:       "",              // 空字符串，x-ui API 会使用默认配置
		StreamSettings: "",              // 空字符串，x-ui API 会使用默认配置
		Sniffing:       "",              // 空字符串，x-ui API 会使用默认配置
		Listen:         "",              // 默认空字符串
	}

	// // 从订单中获取过期时间和流量限制（如果订单有这些信息）
	// if order != nil {
	// 	if order.ExpiresAt > 0 {
	// 		xuiReq.ExpiryTime = order.ExpiresAt
	// 	}
	// 	// 注意：订单模型中可能没有 Total 字段，这里只是示例
	// 	// 如果订单需要配置流量限制，需要在订单模型中添加相应字段
	// }

	return xuiReq, nil
}

// 订单信息结构（用于从数据库查询，与x-ui中的model.Order保持一致）
type OrderInfo struct {
	Id        int    `gorm:"primaryKey"`
	OrderId   string `gorm:"uniqueIndex;column:order_id"`
	UserId    string `gorm:"column:user_id"`
	Status    string `gorm:"column:status"`
	Amount    int64  `gorm:"column:amount"`
	PaidAt    int64  `gorm:"column:paid_at"`
	ExpiresAt int64  `gorm:"column:expires_at"`
	InboundId int    `gorm:"column:inbound_id"`
	CreatedAt int64  `gorm:"column:created_at"`
	UsedAt    int64  `gorm:"column:used_at"`
	Remark    string `gorm:"column:remark"`
}

// TableName 指定表名为 orders（与 x-ui 中的表名保持一致）
func (OrderInfo) TableName() string {
	return "orders"
}

// 获取订单信息（从数据库查询）
func getOrderInfo(orderId string) (*OrderInfo, error) {
	// 检查数据库文件是否存在
	if _, err := os.Stat(config.XUIDBPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("数据库文件不存在")
	}

	// 连接数据库
	db, err := gorm.Open(sqlite.Open(config.XUIDBPath), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	var order OrderInfo
	err = db.Where("order_id = ?", orderId).First(&order).Error
	if err != nil {
		return nil, fmt.Errorf("订单不存在: %v", err)
	}

	return &order, nil
}

// createOrUpdateOrderTest 测试接口：创建或更新订单（通过浏览器访问）
// 访问示例: http://localhost:8080/test/create-order?orderId=ORDER_001&userId=USER_001&expiresAt=1735689600000
// 参数说明:
//   - orderId: 订单号（必填）
//   - userId: 用户ID（必填）
//   - expiresAt: 过期时间（毫秒时间戳，可选，默认为0表示永不过期）
func createOrUpdateOrderTest(c *gin.Context) {
	// 从URL查询参数获取数据
	orderId := c.Query("orderId")
	userId := c.Query("userId")
	expiresAtStr := c.Query("expiresAt")

	// 参数验证
	if orderId == "" {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Msg:     "参数错误: orderId 不能为空",
		})
		return
	}

	if userId == "" {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Msg:     "参数错误: userId 不能为空",
		})
		return
	}

	// 解析过期时间
	var expiresAt int64 = 0 // 默认为0（永不过期）
	if expiresAtStr != "" {
		var err error
		expiresAt, err = strconv.ParseInt(expiresAtStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, Response{
				Success: false,
				Msg:     fmt.Sprintf("参数错误: expiresAt 必须是有效的时间戳，错误: %v", err),
			})
			return
		}
	}

	// 检查数据库文件是否存在
	if _, err := os.Stat(config.XUIDBPath); os.IsNotExist(err) {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Msg:     fmt.Sprintf("数据库文件不存在: %s", config.XUIDBPath),
		})
		return
	}

	// 连接数据库
	db, err := gorm.Open(sqlite.Open(config.XUIDBPath), &gorm.Config{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Msg:     fmt.Sprintf("连接数据库失败: %v", err),
		})
		return
	}
	defer func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	// 检查订单是否已存在
	var existingOrder OrderInfo
	result := db.Where("order_id = ?", orderId).First(&existingOrder)

	now := time.Now().Unix() * 1000 // 毫秒时间戳

	if result.Error == nil {
		// 订单已存在，更新订单
		updates := map[string]interface{}{
			"user_id":    userId,
			"expires_at": expiresAt,
		}

		// 如果订单状态不是 paid，设置为 paid（方便测试）
		if existingOrder.Status != "paid" {
			updates["status"] = "paid"
			if existingOrder.PaidAt == 0 {
				updates["paid_at"] = now
			}
		}

		err = db.Model(&OrderInfo{}).Where("order_id = ?", orderId).Updates(updates).Error
		if err != nil {
			c.JSON(http.StatusInternalServerError, Response{
				Success: false,
				Msg:     fmt.Sprintf("更新订单失败: %v", err),
			})
			return
		}

		c.JSON(http.StatusOK, Response{
			Success: true,
			Msg:     fmt.Sprintf("订单已更新: %s", orderId),
			Data: map[string]interface{}{
				"orderId":   orderId,
				"userId":    userId,
				"expiresAt": expiresAt,
				"status":    "paid",
				"action":    "updated",
			},
		})
		return
	}

	// 订单不存在，创建新订单
	newOrder := OrderInfo{
		OrderId:   orderId,
		UserId:    userId,
		Status:    "paid", // 默认设为已支付，方便测试
		Amount:    0,
		ExpiresAt: expiresAt,
		PaidAt:    now,
		CreatedAt: now / 1000, // CreatedAt 使用秒级时间戳
	}

	err = db.Create(&newOrder).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Msg:     fmt.Sprintf("创建订单失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Msg:     fmt.Sprintf("订单已创建: %s", orderId),
		Data: map[string]interface{}{
			"orderId":   orderId,
			"userId":    userId,
			"expiresAt": expiresAt,
			"status":    "paid",
			"action":    "created",
		},
	})
}

// 生成备注名称
func generateRemark(userId, orderId string) string {
	// 格式：用户节点-订单号后4位
	if len(orderId) > 4 {
		return fmt.Sprintf("用户节点-%s", orderId[len(orderId)-4:])
	}
	return fmt.Sprintf("用户节点-%s", orderId)
}

// 查询订单状态代理
func getOrderStatusProxy(c *gin.Context) {
	// 从客户端请求体中解析 OrderId
	// 客户端需要发送: {"orderId": "ORDER_123"}
	var req OrderStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 如果请求体中没有 orderId 或格式错误，会在这里返回错误
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Msg:     "请求参数错误: " + err.Error(),
		})
		return
	}

	// 此时 req.OrderId 已经从请求体中解析出来了
	// 将 req（包含 OrderId）传递给 x-ui API
	response, err := callXUIAPI("/api/v1/order/status", req)
	if err != nil {
		log.Printf("调用x-ui API失败: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Msg:     "查询订单状态失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// 调用x-ui API（带签名）
func callXUIAPI(path string, payload interface{}) (*Response, error) {
	// 序列化请求体
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %v", err)
	}
	bodyStr := string(bodyBytes)

	// 生成签名所需的参数
	timestamp := time.Now().Unix()
	nonce := generateNonce()

	// 生成签名
	signature := generateSignature(timestamp, nonce, bodyStr, config.APISecret)

	// 构建请求
	url := config.XUIServerURL + path
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", config.APIKey)
	req.Header.Set("X-Timestamp", strconv.FormatInt(timestamp, 10))
	req.Header.Set("X-Nonce", nonce)
	req.Header.Set("X-Signature", signature)

	// 发送请求
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	// 解析响应（x-ui 使用 obj 字段，我们需要转换为 data）
	var xuiResponse struct {
		Success bool        `json:"success"`
		Msg     string      `json:"msg"`
		Obj     interface{} `json:"obj"`
	}
	if err := json.Unmarshal(respBody, &xuiResponse); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		return &Response{
			Success: false,
			Msg:     fmt.Sprintf("x-ui API返回错误: %s (HTTP %d)", xuiResponse.Msg, resp.StatusCode),
		}, nil
	}

	// 将 x-ui 的 obj 字段转换为 data 字段
	response := Response{
		Success: xuiResponse.Success,
		Msg:     xuiResponse.Msg,
		Data:    xuiResponse.Obj,
	}

	return &response, nil
}

// 生成签名
func generateSignature(timestamp int64, nonce, body, secret string) string {
	signStr := fmt.Sprintf("%d%s%s%s", timestamp, nonce, body, secret)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signStr))
	return hex.EncodeToString(mac.Sum(nil))
}

// 生成随机nonce
func generateNonce() string {
	return fmt.Sprintf("%d%d", time.Now().UnixNano(), time.Now().Unix()%10000)
}

// 获取环境变量
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// 创建或获取API Key
func createOrGetAPIKey() (string, string, error) {
	// 检查数据库文件是否存在
	if _, err := os.Stat(config.XUIDBPath); os.IsNotExist(err) {
		return "", "", fmt.Errorf("x-ui数据库文件不存在: %s", config.XUIDBPath)
	}

	// 连接数据库
	db, err := gorm.Open(sqlite.Open(config.XUIDBPath), &gorm.Config{})
	if err != nil {
		return "", "", fmt.Errorf("连接数据库失败: %v", err)
	}

	// 检查表是否存在，如果不存在则创建（依赖 x-ui 服务创建表，这里只是兜底）
	// 注意：正常情况下，x-ui 服务启动时会通过 database.InitDB 创建所有表
	// 这里只是为了防止 x-ui 服务未启动时 backend-proxy 先启动的情况
	var count int64
	err = db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='api_keys'").Scan(&count).Error
	if err == nil && count == 0 {
		// 表不存在，创建表（仅在必要时）
		err = db.AutoMigrate(&APIKey{})
		if err != nil {
			return "", "", fmt.Errorf("创建 api_keys 表失败: %v", err)
		}
		log.Println("⚠️  检测到 api_keys 表不存在，已自动创建。建议先启动 x-ui 服务初始化数据库。")
	}

	// 检查是否已存在用于后端代理的API Key
	var existingKey APIKey
	result := db.Where("name LIKE ?", "%后端代理%").First(&existingKey)

	if result.Error == nil {
		// 找到了现有的API Key，更新.env文件
		log.Printf("找到现有的API Key: %s", existingKey.Key)
		err = updateEnvFile(existingKey.Key, existingKey.Secret)
		if err != nil {
			log.Printf("警告: 更新.env文件失败: %v", err)
		}
		return existingKey.Key, existingKey.Secret, nil
	}

	// 创建新的API Key
	apiKey := fmt.Sprintf("xui_%s", randomString(32))
	apiSecret := randomString(64)

	newAPIKey := APIKey{
		Key:        apiKey,
		Secret:     apiSecret,
		Name:       "Android客户端后端代理",
		Status:     "active",
		CreatedAt:  time.Now().Unix(),
		RateLimit:  100,
		AllowedIPs: "",
	}

	err = db.Create(&newAPIKey).Error
	if err != nil {
		return "", "", fmt.Errorf("创建API Key失败: %v", err)
	}

	// 更新.env文件
	err = updateEnvFile(apiKey, apiSecret)
	if err != nil {
		log.Printf("警告: 更新.env文件失败: %v", err)
	}

	log.Printf("✅ 创建新的API Key: %s", apiKey)
	return apiKey, apiSecret, nil
}

// 更新.env文件
func updateEnvFile(apiKey, apiSecret string) error {
	envPath := ".env"

	// 如果.env文件不存在，创建它
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		// 从env.example复制
		examplePath := "env.example"
		if _, err := os.Stat(examplePath); err == nil {
			input, err := os.ReadFile(examplePath)
			if err == nil {
				os.WriteFile(envPath, input, 0644)
			}
		}
	}

	// 读取.env文件
	file, err := os.Open(envPath)
	if err != nil {
		return fmt.Errorf("无法打开.env文件: %v", err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	keyUpdated := false
	secretUpdated := false

	for scanner.Scan() {
		line := scanner.Text()

		// 更新XUI_API_KEY
		if strings.HasPrefix(line, "XUI_API_KEY=") {
			lines = append(lines, fmt.Sprintf("XUI_API_KEY=%s", apiKey))
			keyUpdated = true
			continue
		}

		// 更新XUI_API_SECRET
		if strings.HasPrefix(line, "XUI_API_SECRET=") {
			lines = append(lines, fmt.Sprintf("XUI_API_SECRET=%s", apiSecret))
			secretUpdated = true
			continue
		}

		lines = append(lines, line)
	}

	// 如果不存在，添加新行
	if !keyUpdated {
		lines = append(lines, fmt.Sprintf("XUI_API_KEY=%s", apiKey))
	}
	if !secretUpdated {
		lines = append(lines, fmt.Sprintf("XUI_API_SECRET=%s", apiSecret))
	}

	// 写入文件
	output := strings.Join(lines, "\n")
	err = os.WriteFile(envPath, []byte(output), 0644)
	if err != nil {
		return fmt.Errorf("写入.env文件失败: %v", err)
	}

	log.Printf("✅ 已更新.env文件中的API Key和Secret")
	return nil
}

// 生成随机字符串
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// CORS中间件
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
