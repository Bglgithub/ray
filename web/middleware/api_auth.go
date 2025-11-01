package middleware

import (
	"bytes"
	"fmt"
	"io"
	"sync"
	"time"
	"x-ui/logger"
	"x-ui/web/controller"
	"x-ui/web/service"

	"github.com/gin-gonic/gin"
)

type RateLimiter struct {
	requests map[string][]int64
	mu       sync.RWMutex
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]int64),
	}
}

// CheckRateLimit 检查速率限制
func (rl *RateLimiter) CheckRateLimit(key string, limit int) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now().Unix()
	
	// 清理一分钟前的记录
	if requests, ok := rl.requests[key]; ok {
		validRequests := make([]int64, 0)
		for _, timestamp := range requests {
			if now-timestamp < 60 {
				validRequests = append(validRequests, timestamp)
			}
		}
		rl.requests[key] = validRequests
		
		// 检查是否超过限制
		if len(validRequests) >= limit {
			return false
		}
	}

	// 添加当前请求
	rl.requests[key] = append(rl.requests[key], now)
	return true
}

var globalRateLimiter = NewRateLimiter()

// APIAuthMiddleware API认证中间件
func APIAuthMiddleware() gin.HandlerFunc {
	apiKeyService := &service.APIKeyService{}
	
	return func(c *gin.Context) {
		// 获取API Key
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			controller.PureJsonMsg(c, false, "缺少API密钥")
			c.Abort()
			return
		}

		// 获取客户端IP
		clientIP := controller.GetRemoteIp(c)
		
		// 验证API Key
		keyInfo, err := apiKeyService.VerifyAPIKey(apiKey, clientIP)
		if err != nil {
			logger.Warningf("API Key verification failed: %s, IP: %s, Error: %v", apiKey, clientIP, err)
			controller.PureJsonMsg(c, false, "API密钥验证失败: "+err.Error())
			c.Abort()
			return
		}

		// 速率限制检查
		if !globalRateLimiter.CheckRateLimit(apiKey, keyInfo.RateLimit) {
			logger.Warningf("Rate limit exceeded for API Key: %s, IP: %s", apiKey, clientIP)
			controller.PureJsonMsg(c, false, "请求过于频繁，请稍后再试")
			c.Abort()
			return
		}

		// 签名验证
		timestampStr := c.GetHeader("X-Timestamp")
		nonce := c.GetHeader("X-Nonce")
		signature := c.GetHeader("X-Signature")

		if timestampStr == "" || nonce == "" || signature == "" {
			controller.PureJsonMsg(c, false, "缺少必要的签名参数")
			c.Abort()
			return
		}

		var timestamp int64
		_, err = fmt.Sscanf(timestampStr, "%d", &timestamp)
		if err != nil {
			controller.PureJsonMsg(c, false, "无效的时间戳格式")
			c.Abort()
			return
		}

		// 读取请求体（用于签名验证）
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			controller.PureJsonMsg(c, false, "读取请求体失败")
			c.Abort()
			return
		}
		// 恢复请求体（供后续处理使用）
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		bodyStr := string(bodyBytes)

		// 验证签名
		err = apiKeyService.VerifySignature(keyInfo, timestamp, nonce, signature, bodyStr)
		if err != nil {
			logger.Warningf("Signature verification failed: %s, IP: %s, Error: %v", apiKey, clientIP, err)
			controller.PureJsonMsg(c, false, "签名验证失败: "+err.Error())
			c.Abort()
			return
		}

		// 更新API Key最后使用时间（异步，不阻塞请求）
		go func() {
			err := apiKeyService.UpdateAPIKeyLastUsed(apiKey)
			if err != nil {
				logger.Warning("Failed to update API key last used time:", err)
			}
		}()

		// 将API Key信息存储到上下文中
		c.Set("api_key", keyInfo)

		c.Next()
	}
}

