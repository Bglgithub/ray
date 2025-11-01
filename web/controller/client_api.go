package controller

import (
	"crypto/rand"
	"fmt"
	"x-ui/database/model"
	"x-ui/logger"
	"x-ui/util/random"
	"x-ui/web/service"

	"github.com/gin-gonic/gin"
)

type ClientAPIController struct {
	BaseController
	inboundService service.InboundService
	orderService   service.OrderService
	xrayService    service.XrayService
}

type CreateInboundRequest struct {
	OrderId    string `json:"orderId" binding:"required"`  // 订单号（必填）
	UserId     string `json:"userId" binding:"required"`   // 用户ID（必填）
	Protocol   string `json:"protocol" binding:"required"` // 协议类型
	Port       int    `json:"port"`                        // 端口（可选，不填则自动分配）
	Remark     string `json:"remark"`                      // 备注
	ExpiryTime int64  `json:"expiryTime"`                  // 过期时间（时间戳，毫秒）
	Total      int64  `json:"total"`                       // 总流量限制（字节）
	// 以下字段可选，如果不提供则使用默认配置
	Settings       string `json:"settings"`       // 协议配置（JSON字符串）
	StreamSettings string `json:"streamSettings"` // 传输配置（JSON字符串）
	Sniffing       string `json:"sniffing"`       // 流量嗅探配置（JSON字符串）
	Listen         string `json:"listen"`         // 监听地址
}

type CreateInboundResponse struct {
	Inbound *model.Inbound `json:"inbound"` // 完整的入站对象
}

func NewClientAPIController(g *gin.RouterGroup) *ClientAPIController {
	a := &ClientAPIController{
		inboundService: service.InboundService{},
		orderService:   service.OrderService{},
	}
	a.initRouter(g)
	return a
}

func (a *ClientAPIController) initRouter(g *gin.RouterGroup) {
	// 路由由 web.go 中配置，使用中间件保护
}

// CreateInbound 创建入站配置（安全的API接口）
func (a *ClientAPIController) CreateInbound(c *gin.Context) {
	var req CreateInboundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pureJsonMsg(c, false, "请求参数错误: "+err.Error())
		return
	}

	// 1. 验证订单
	order, err := a.orderService.VerifyOrder(req.OrderId)
	if err != nil {
		pureJsonMsg(c, false, "订单验证失败: "+err.Error())
		return
	}

	// 2. 验证订单用户ID是否匹配（防止订单被其他用户使用）
	if order.UserId != req.UserId {
		logger.Warningf("Order user mismatch: orderId=%s, orderUserId=%s, requestUserId=%s",
			req.OrderId, order.UserId, req.UserId)
		pureJsonMsg(c, false, "订单用户不匹配")
		return
	}

	// 3. 检查订单是否已被使用（双重检查）
	if order.Status == "used" && order.InboundId > 0 {
		// 订单已使用，返回已创建的入站信息
		inbound, err := a.inboundService.GetInbound(order.InboundId)
		if err == nil {
			jsonObj(c, CreateInboundResponse{
				Inbound: inbound,
			}, nil)
			return
		}
	}

	// 4. 验证协议类型
	validProtocols := map[string]bool{
		"vmess":       true,
		"vless":       true,
		"trojan":      true,
		"shadowsocks": true,
	}
	if !validProtocols[req.Protocol] {
		pureJsonMsg(c, false, "不支持的协议类型: "+req.Protocol)
		return
	}

	// 5. 端口处理：如果没有指定端口，自动分配一个可用端口
	port := req.Port
	if port == 0 {
		port = a.allocatePort()
		if port == 0 {
			pureJsonMsg(c, false, "无法分配可用端口")
			return
		}
	} else {
		// 检查端口是否已被占用
		exist, err := a.inboundService.CheckPortExist(port, 0)
		if err != nil {
			pureJsonMsg(c, false, "检查端口失败: "+err.Error())
			return
		}
		if exist {
			pureJsonMsg(c, false, fmt.Sprintf("端口 %d 已被占用", port))
			return
		}
	}

	// 6. 创建入站配置
	inbound := &model.Inbound{
		UserId:     1, // 使用系统默认用户ID
		Port:       port,
		Protocol:   model.Protocol(req.Protocol),
		Remark:     req.Remark,
		Enable:     true,
		ExpiryTime: req.ExpiryTime,
		Total:      req.Total,
		Listen:     req.Listen,
		Tag:        fmt.Sprintf("inbound-%v", port),
	}

	// 使用提供的配置或默认配置
	if req.Settings != "" {
		inbound.Settings = req.Settings
	} else {
		// 使用默认配置（这里应该根据协议生成默认配置）
		inbound.Settings = a.getDefaultSettings(req.Protocol)
	}

	if req.StreamSettings != "" {
		inbound.StreamSettings = req.StreamSettings
	} else {
		inbound.StreamSettings = "{}" // 默认空配置
	}

	if req.Sniffing != "" {
		inbound.Sniffing = req.Sniffing
	} else {
		inbound.Sniffing = `{"enabled":true,"destOverride":["http","tls"]}`
	}

	// 7. 保存入站配置
	err = a.inboundService.AddInbound(inbound)
	if err != nil {
		logger.Error("Failed to add inbound:", err)
		pureJsonMsg(c, false, "创建入站配置失败: "+err.Error())
		return
	}

	// 8. 标记订单为已使用
	err = a.orderService.MarkOrderAsUsed(req.OrderId, inbound.Id)
	if err != nil {
		logger.Error("Failed to mark order as used:", err)
		// 即使标记失败，入站已创建成功，不影响返回结果
	}

	// 9. 标记需要重启xray
	a.xrayService = service.XrayService{}
	a.xrayService.SetToNeedRestart()

	// 10. 重新从数据库获取完整的入站对象（确保包含所有字段）
	createdInbound, err := a.inboundService.GetInbound(inbound.Id)
	if err != nil {
		logger.Error("Failed to get created inbound:", err)
		pureJsonMsg(c, false, "创建成功但获取入站信息失败: "+err.Error())
		return
	}

	logger.Infof("Inbound created via API: orderId=%s, userId=%s, inboundId=%d, port=%d",
		req.OrderId, req.UserId, createdInbound.Id, createdInbound.Port)

	// 11. 返回完整的入站对象
	jsonObj(c, CreateInboundResponse{
		Inbound: createdInbound,
	}, nil)
}

// GetOrderStatus 查询订单状态
func (a *ClientAPIController) GetOrderStatus(c *gin.Context) {
	type StatusRequest struct {
		OrderId string `json:"orderId" binding:"required"`
	}

	var req StatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pureJsonMsg(c, false, "请求参数错误: "+err.Error())
		return
	}

	order, err := a.orderService.GetOrderByOrderId(req.OrderId)
	if err != nil {
		pureJsonMsg(c, false, "订单不存在")
		return
	}

	type OrderStatusResponse struct {
		OrderId   string `json:"orderId"`
		Status    string `json:"status"`
		InboundId int    `json:"inboundId"`
		PaidAt    int64  `json:"paidAt"`
		UsedAt    int64  `json:"usedAt"`
	}

	jsonObj(c, OrderStatusResponse{
		OrderId:   order.OrderId,
		Status:    order.Status,
		InboundId: order.InboundId,
		PaidAt:    order.PaidAt,
		UsedAt:    order.UsedAt,
	}, nil)
}

// allocatePort 自动分配一个可用端口（从10000开始查找）
func (a *ClientAPIController) allocatePort() int {
	startPort := 10000
	endPort := 65535

	for port := startPort; port <= endPort; port++ {
		exist, err := a.inboundService.CheckPortExist(port, 0)
		if err == nil && !exist {
			return port
		}
	}
	return 0
}

// getDefaultSettings 根据协议返回默认配置
func (a *ClientAPIController) getDefaultSettings(protocol string) string {
	switch protocol {
	case "vmess":
		return `{"clients":[{"id":"` + generateUUID() + `","alterId":0}],"disableInsecureEncryption":false}`
	case "vless":
		return `{"clients":[{"id":"` + generateUUID() + `","flow":""}],"decryption":"none"}`
	case "trojan":
		return `{"clients":[{"password":"` + generateRandomPassword() + `"}]}`
	case "shadowsocks":
		return `{"method":"aes-256-gcm","password":"` + generateRandomPassword() + `"}`
	default:
		return "{}"
	}
}

// generateUUID 生成UUID v4格式
func generateUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // Version 4
	b[8] = (b[8] & 0x3f) | 0x80 // Variant is 10
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// generateRandomPassword 生成随机密码
func generateRandomPassword() string {
	return random.Seq(16)
}
