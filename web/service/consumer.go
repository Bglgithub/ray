package service

import (
	"x-ui/database"
	"x-ui/database/model"

	"gorm.io/gorm"
)

type ConsumerService struct {
}

// GetConsumerList 获取消费者列表
func (s *ConsumerService) GetConsumerList(limit int) ([]*model.Consumer, error) {
	db := database.GetDB()
	var consumers []*model.Consumer
	query := db.Model(&model.Consumer{}).Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&consumers).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return consumers, nil
}

// GetConsumerById 根据ID获取消费者
func (s *ConsumerService) GetConsumerById(id int) (*model.Consumer, error) {
	db := database.GetDB()
	consumer := &model.Consumer{}
	err := db.Where("id = ?", id).First(consumer).Error
	if err != nil {
		return nil, err
	}
	return consumer, nil
}

