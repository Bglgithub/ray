package controller

import (
	"github.com/gin-gonic/gin"
)

type XUIController struct {
	BaseController

	inboundController    *InboundController
	settingController    *SettingController
	serverSiteController *ServerSiteController
	orderController      *OrderController
	consumerController   *ConsumerController
}

func NewXUIController(g *gin.RouterGroup) *XUIController {
	a := &XUIController{}
	a.initRouter(g)
	return a
}

func (a *XUIController) initRouter(g *gin.RouterGroup) {
	g = g.Group("/xui")
	g.Use(a.checkLogin)

	g.GET("/", a.index)
	g.GET("/inbounds", a.inbounds)
	g.GET("/servers", a.servers)
	g.GET("/orders", a.orders)
	g.GET("/consumers", a.consumers)
	g.GET("/setting", a.setting)

	a.inboundController = NewInboundController(g)
	a.settingController = NewSettingController(g)
	a.serverSiteController = NewServerSiteController(g)
	a.orderController = NewOrderController(g)
	a.consumerController = NewConsumerController(g)
}

func (a *XUIController) index(c *gin.Context) {
	html(c, "index.html", "系统状态", nil)
}

func (a *XUIController) inbounds(c *gin.Context) {
	html(c, "inbounds.html", "入站列表", nil)
}

func (a *XUIController) servers(c *gin.Context) {
	html(c, "servers.html", "服务器列表", nil)
}

func (a *XUIController) orders(c *gin.Context) {
	html(c, "orders.html", "订单列表", nil)
}

func (a *XUIController) consumers(c *gin.Context) {
	html(c, "consumers.html", "用户列表", nil)
}

func (a *XUIController) setting(c *gin.Context) {
	html(c, "setting.html", "设置", nil)
}
