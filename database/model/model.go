package model

import (
	"fmt"
	"x-ui/util/json_util"
	"x-ui/xray"
)

type Protocol string

const (
	VMess       Protocol = "vmess"
	VLESS       Protocol = "vless"
	Dokodemo    Protocol = "Dokodemo-door"
	Http        Protocol = "http"
	Trojan      Protocol = "trojan"
	Shadowsocks Protocol = "shadowsocks"
)

// 这里的user指的是admin用户
type User struct {
	Id       int    `json:"id" gorm:"primaryKey;autoIncrement"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// 这里的consumer指的是消费者用户
type Consumer struct {
	Id            int   `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId        int   `json:"userId"`
	CreatedAt     int64 `json:"createdAt" gorm:"autoCreateTime"` // 创建时间
	LastLoginAt   int64 `json:"LastLoginAt"`                     //  最后登录时间
	FreeExpiredAt int64 `json:"expiredAt"`                       // 免费试用过期时间，0表示过期
	FromFlag      int   `json:"fromFlag"`                        // 来源标志
}

type Inbound struct {
	Id         int    `json:"id" form:"id" gorm:"primaryKey;autoIncrement"` // 入站ID
	UserId     int    `json:"-"`                                            // 用户ID
	Up         int64  `json:"up" form:"up"`                                 // 上传流量
	Down       int64  `json:"down" form:"down"`                             // 下载流量
	Total      int64  `json:"total" form:"total"`                           // 总流量
	Remark     string `json:"remark" form:"remark"`                         // 备注
	Enable     bool   `json:"enable" form:"enable"`                         // 是否启用
	ExpiryTime int64  `json:"expiryTime" form:"expiryTime"`                 // 过期时间

	// config part
	Listen         string   `json:"listen" form:"listen"`                 // 监听地址
	Port           int      `json:"port" form:"port" gorm:"unique"`       // 端口
	Protocol       Protocol `json:"protocol" form:"protocol"`             // 协议
	Settings       string   `json:"settings" form:"settings"`             // 设置（JSON字符串）
	StreamSettings string   `json:"streamSettings" form:"streamSettings"` // 流设置（JSON字符串）
	Tag            string   `json:"tag" form:"tag" gorm:"unique"`         // 标签
	Sniffing       string   `json:"sniffing" form:"sniffing"`             // 嗅探（JSON字符串）
}

func (i *Inbound) GenXrayInboundConfig() *xray.InboundConfig {
	listen := i.Listen
	if listen != "" {
		listen = fmt.Sprintf("\"%v\"", listen)
	}
	return &xray.InboundConfig{
		Listen:         json_util.RawMessage(listen),
		Port:           i.Port,
		Protocol:       string(i.Protocol),
		Settings:       json_util.RawMessage(i.Settings),
		StreamSettings: json_util.RawMessage(i.StreamSettings),
		Tag:            i.Tag,
		Sniffing:       json_util.RawMessage(i.Sniffing),
	}
}

type Setting struct {
	Id    int    `json:"id" form:"id" gorm:"primaryKey;autoIncrement"`
	Key   string `json:"key" form:"key"`
	Value string `json:"value" form:"value"`
}

// APIKey 用于客户端API认证
type APIKey struct {
	Id         int    `json:"id" gorm:"primaryKey;autoIncrement"`
	Key        string `json:"key" gorm:"uniqueIndex;not null"` // API密钥
	Secret     string `json:"-" gorm:"not null"`               // API密钥对应的密钥（用于签名验证）
	Name       string `json:"name"`                            // 密钥名称/描述
	Status     string `json:"status" gorm:"default:active"`    // 状态：active/inactive
	CreatedAt  int64  `json:"createdAt" gorm:"autoCreateTime"` // 创建时间
	LastUsedAt int64  `json:"lastUsedAt"`                      // 最后使用时间
	RateLimit  int    `json:"rateLimit" gorm:"default:100"`    // 每分钟请求限制
	AllowedIPs string `json:"allowedIPs"`                      // 允许的IP地址，逗号分隔
}

// Order 订单记录，用于验证付费
type Order struct {
	Id        int    `json:"id" gorm:"primaryKey;autoIncrement"`
	OrderId   string `json:"orderId" gorm:"uniqueIndex;not null"` // 订单号（唯一）
	UserId    string `json:"userId" gorm:"index;not null"`        // 用户ID（来自Android客户端）
	Status    string `json:"status" gorm:"default:pending"`       // 订单状态：pending/paid/used/expired
	Amount    int64  `json:"amount"`                              // 订单金额（分）
	PaidAt    int64  `json:"paidAt"`                              // 支付时间
	ExpiresAt int64  `json:"expiresAt"`                           // 订单过期时间
	InboundId int    `json:"inboundId"`                           // 关联的入站ID（如果已创建）
	CreatedAt int64  `json:"createdAt" gorm:"autoCreateTime"`     // 创建时间
	UsedAt    int64  `json:"usedAt"`                              // 使用时间（创建入站时）
	Remark    string `json:"remark"`                              // 备注
}
