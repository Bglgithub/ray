package service

import (
	"errors"
	"strings"
	"time"
	"x-ui/database"
	"x-ui/database/model"
	"x-ui/logger"
)

type OrderService struct {
}

// CreateOrder 创建订单
func (s *OrderService) CreateOrder(orderId string, userId string, amount int64, expiresAt int64) (*model.Order, error) {
	order := &model.Order{
		OrderId:   orderId,
		UserId:    userId,
		Status:    "pending",
		Amount:    amount,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now().Unix(),
	}

	db := database.GetDB()
	err := db.Create(order).Error
	if err != nil {
		// 检查是否是唯一约束冲突（SQLite返回的错误）
		errStr := err.Error()
		if strings.Contains(errStr, "UNIQUE constraint failed") || 
		   strings.Contains(errStr, "duplicate key") {
			return nil, errors.New("订单号已存在")
		}
		return nil, err
	}

	return order, nil
}

// GetOrderByOrderId 根据订单号获取订单
func (s *OrderService) GetOrderByOrderId(orderId string) (*model.Order, error) {
	db := database.GetDB()
	order := &model.Order{}
	err := db.Where("order_id = ?", orderId).First(order).Error
	if err != nil {
		return nil, err
	}
	return order, nil
}

// VerifyOrder 验证订单是否可以用于创建入站
func (s *OrderService) VerifyOrder(orderId string) (*model.Order, error) {
	order, err := s.GetOrderByOrderId(orderId)
	if err != nil {
		return nil, errors.New("订单不存在")
	}

	now := time.Now().Unix() * 1000 // 毫秒时间戳

	// 检查订单状态
	if order.Status == "used" {
		return nil, errors.New("订单已被使用")
	}

	if order.Status == "expired" {
		return nil, errors.New("订单已过期")
	}

	if order.Status != "paid" {
		return nil, errors.New("订单尚未支付")
	}

	// 检查订单是否过期
	if order.ExpiresAt > 0 && order.ExpiresAt < now {
		// 更新订单状态为过期
		s.UpdateOrderStatus(orderId, "expired")
		return nil, errors.New("订单已过期")
	}

	return order, nil
}

// UpdateOrderStatus 更新订单状态
func (s *OrderService) UpdateOrderStatus(orderId string, status string) error {
	updates := map[string]interface{}{
		"status": status,
	}

	if status == "paid" {
		updates["paid_at"] = time.Now().Unix() * 1000
	} else if status == "used" {
		updates["used_at"] = time.Now().Unix() * 1000
	}

	db := database.GetDB()
	return db.Model(&model.Order{}).
		Where("order_id = ?", orderId).
		Updates(updates).Error
}

// MarkOrderAsUsed 标记订单为已使用
func (s *OrderService) MarkOrderAsUsed(orderId string, inboundId int) error {
	db := database.GetDB()
	return db.Model(&model.Order{}).
		Where("order_id = ?", orderId).
		Updates(map[string]interface{}{
			"status":     "used",
			"inbound_id": inboundId,
			"used_at":    time.Now().Unix() * 1000,
		}).Error
}

// GetOrderList 获取订单列表
func (s *OrderService) GetOrderList(userId string, limit int) ([]*model.Order, error) {
	db := database.GetDB()
	var orders []*model.Order
	query := db.Model(&model.Order{}).Order("created_at DESC")
	
	if userId != "" {
		query = query.Where("user_id = ?", userId)
	}
	
	if limit > 0 {
		query = query.Limit(limit)
	}
	
	err := query.Find(&orders).Error
	return orders, err
}

// CleanupExpiredOrders 清理过期订单（定时任务）
func (s *OrderService) CleanupExpiredOrders() error {
	now := time.Now().Unix() * 1000
	db := database.GetDB()
	result := db.Model(&model.Order{}).
		Where("expires_at > 0 AND expires_at < ? AND status IN ?", now, []string{"pending", "paid"}).
		Update("status", "expired")
	
	if result.Error != nil {
		return result.Error
	}
	
	if result.RowsAffected > 0 {
		logger.Infof("Cleaned up %d expired orders", result.RowsAffected)
	}
	
	return nil
}

