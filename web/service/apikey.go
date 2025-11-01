package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
	"x-ui/database"
	"x-ui/database/model"
	"x-ui/logger"
	"x-ui/util/random"
)

type APIKeyService struct {
}

// CreateAPIKey 创建新的API密钥
func (s *APIKeyService) CreateAPIKey(name string, rateLimit int, allowedIPs string) (*model.APIKey, error) {
	if rateLimit <= 0 {
		rateLimit = 100 // 默认限制
	}

	apiKey := &model.APIKey{
		Key:        fmt.Sprintf("xui_%s", random.Seq(32)),
		Secret:     random.Seq(64),
		Name:       name,
		Status:     "active",
		RateLimit:  rateLimit,
		AllowedIPs: allowedIPs,
		CreatedAt:  time.Now().Unix(),
	}

	db := database.GetDB()
	err := db.Create(apiKey).Error
	if err != nil {
		return nil, err
	}

	logger.Info("API Key created:", apiKey.Key)
	return apiKey, nil
}

// GetAPIKey 根据Key获取API密钥信息
func (s *APIKeyService) GetAPIKey(key string) (*model.APIKey, error) {
	db := database.GetDB()
	apiKey := &model.APIKey{}
	err := db.Where("key = ?", key).First(apiKey).Error
	if err != nil {
		return nil, err
	}
	return apiKey, nil
}

// UpdateAPIKeyLastUsed 更新API密钥最后使用时间
func (s *APIKeyService) UpdateAPIKeyLastUsed(key string) error {
	db := database.GetDB()
	return db.Model(&model.APIKey{}).
		Where("key = ?", key).
		Update("last_used_at", time.Now().Unix()).Error
}

// VerifyAPIKey 验证API密钥是否有效
func (s *APIKeyService) VerifyAPIKey(key string, ip string) (*model.APIKey, error) {
	apiKey, err := s.GetAPIKey(key)
	if err != nil {
		return nil, errors.New("无效的API密钥")
	}

	if apiKey.Status != "active" {
		return nil, errors.New("API密钥已被禁用")
	}

	// 检查IP白名单
	if apiKey.AllowedIPs != "" {
		allowedIPs := strings.Split(apiKey.AllowedIPs, ",")
		ipAllowed := false
		for _, allowedIP := range allowedIPs {
			allowedIP = strings.TrimSpace(allowedIP)
			if allowedIP == ip {
				ipAllowed = true
				break
			}
		}
		if !ipAllowed {
			return nil, errors.New("IP地址不在白名单中")
		}
	}

	return apiKey, nil
}

// VerifySignature 验证请求签名
func (s *APIKeyService) VerifySignature(apiKey *model.APIKey, timestamp int64, nonce string, signature string, body string) error {
	// 检查时间戳（防止重放攻击，允许5分钟内的请求）
	now := time.Now().Unix()
	if now-timestamp > 300 || timestamp-now > 60 {
		return errors.New("请求已过期或时间戳无效")
	}

	// 构建签名字符串: timestamp + nonce + body + secret
	signStr := fmt.Sprintf("%d%s%s%s", timestamp, nonce, body, apiKey.Secret)

	// 计算HMAC-SHA256
	mac := hmac.New(sha256.New, []byte(apiKey.Secret))
	mac.Write([]byte(signStr))
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	// 对比签名（使用常量时间比较防止时序攻击）
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return errors.New("签名验证失败")
	}

	return nil
}

// GetAPIKeyList 获取所有API密钥列表
func (s *APIKeyService) GetAPIKeyList() ([]*model.APIKey, error) {
	db := database.GetDB()
	var apiKeys []*model.APIKey
	err := db.Find(&apiKeys).Error
	return apiKeys, err
}

// UpdateAPIKeyStatus 更新API密钥状态
func (s *APIKeyService) UpdateAPIKeyStatus(key string, status string) error {
	if status != "active" && status != "inactive" {
		return errors.New("无效的状态")
	}
	db := database.GetDB()
	return db.Model(&model.APIKey{}).
		Where("key = ?", key).
		Update("status", status).Error
}

// DeleteAPIKey 删除API密钥
func (s *APIKeyService) DeleteAPIKey(key string) error {
	db := database.GetDB()
	return db.Where("key = ?", key).Delete(&model.APIKey{}).Error
}

