package controller

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"x-ui/web/service"
)

type ConsumerController struct {
	BaseController
	consumerService service.ConsumerService
}

func NewConsumerController(g *gin.RouterGroup) *ConsumerController {
	a := &ConsumerController{}
	a.initRouter(g)
	return a
}

func (a *ConsumerController) initRouter(g *gin.RouterGroup) {
	g = g.Group("/consumer")

	g.POST("/list", a.getConsumers)
}

// 获取用户列表
func (a *ConsumerController) getConsumers(c *gin.Context) {
	// 获取查询参数
	limitStr := c.Query("limit") // 可选：限制数量

	var limit int
	if limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			limit = 0 // 如果解析失败，不限制数量
		}
	}

	consumers, err := a.consumerService.GetConsumerList(limit)
	if err != nil {
		jsonMsg(c, "获取用户列表", err)
		return
	}

	jsonObj(c, consumers, nil)
}

