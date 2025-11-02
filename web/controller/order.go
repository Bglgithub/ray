package controller

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"x-ui/web/service"
)

type OrderController struct {
	BaseController
	orderService service.OrderService
}

func NewOrderController(g *gin.RouterGroup) *OrderController {
	a := &OrderController{}
	a.initRouter(g)
	return a
}

func (a *OrderController) initRouter(g *gin.RouterGroup) {
	g = g.Group("/order")

	g.POST("/list", a.getOrders)
}

// 获取订单列表
func (a *OrderController) getOrders(c *gin.Context) {
	// 获取查询参数
	userId := c.Query("userId") // 可选：按用户ID筛选
	limitStr := c.Query("limit") // 可选：限制数量

	var limit int
	if limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			limit = 0 // 如果解析失败，不限制数量
		}
	}

	orders, err := a.orderService.GetOrderList(userId, limit)
	if err != nil {
		jsonMsg(c, "获取订单列表", err)
		return
	}

	jsonObj(c, orders, nil)
}

