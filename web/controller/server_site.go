package controller

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"x-ui/logger"
	"x-ui/web/entity"

	"github.com/gin-gonic/gin"
)

type ServerSiteController struct {
	BaseController
	backendProxyURL string // backend-proxy 服务地址
}

func NewServerSiteController(g *gin.RouterGroup) *ServerSiteController {
	a := &ServerSiteController{}
	a.initRouter(g)
	// 从环境变量读取 backend-proxy 地址，默认 http://localhost:8080
	// 可以通过设置 BACKEND_PROXY_URL 环境变量来配置
	backendProxyURL := os.Getenv("BACKEND_PROXY_URL")
	if backendProxyURL == "" {
		backendProxyURL = "http://localhost:8080"
	}
	a.backendProxyURL = backendProxyURL
	return a
}

func (a *ServerSiteController) initRouter(g *gin.RouterGroup) {
	g = g.Group("/server-site")

	g.POST("/list", a.getServerSites)
	g.POST("/del/:id", a.deleteServerSite)
}

// 获取服务器列表
func (a *ServerSiteController) getServerSites(c *gin.Context) {
	// 调用 backend-proxy API
	url := a.backendProxyURL + "/api/v1/servers"
	resp, err := http.Get(url)
	if err != nil {
		logger.Error("调用 backend-proxy API 失败:", err)
		jsonMsg(c, "获取服务器列表", fmt.Errorf("无法连接到 backend-proxy 服务 (%s): %v", url, err))
		return
	}
	defer resp.Body.Close()

	// 检查 HTTP 状态码
	if resp.StatusCode != http.StatusOK {
		logger.Error(fmt.Sprintf("backend-proxy 返回错误状态码: %d", resp.StatusCode))
		body, _ := io.ReadAll(resp.Body)
		jsonMsg(c, "获取服务器列表", fmt.Errorf("backend-proxy 返回错误 (HTTP %d): %s", resp.StatusCode, string(body)))
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("读取响应失败:", err)
		jsonMsg(c, "获取服务器列表", err)
		return
	}

	// backend-proxy 返回格式: {"success": true, "msg": "...", "data": [...]}
	// x-ui 期望格式: {"success": true, "msg": "...", "obj": [...]}
	var backendResult struct {
		Success bool        `json:"success"`
		Msg     string      `json:"msg"`
		Data    interface{} `json:"data"`
	}
	if err := json.Unmarshal(body, &backendResult); err != nil {
		logger.Error("解析响应失败:", err, "响应内容:", string(body))
		jsonMsg(c, "获取服务器列表", fmt.Errorf("解析响应失败: %v", err))
		return
	}

	if backendResult.Success {
		// 转换为 x-ui 的格式
		msg := entity.Msg{
			Success: true,
			Msg:     backendResult.Msg,
			Obj:     backendResult.Data,
		}
		c.JSON(http.StatusOK, msg)
	} else {
		jsonMsg(c, backendResult.Msg, fmt.Errorf(backendResult.Msg))
	}
}

// 删除服务器
func (a *ServerSiteController) deleteServerSite(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, "删除", err)
		return
	}

	// 调用 backend-proxy API
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/api/v1/servers/%d", a.backendProxyURL, id), nil)
	if err != nil {
		logger.Error("创建请求失败:", err)
		jsonMsg(c, "删除", err)
		return
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logger.Error("调用 backend-proxy API 失败:", err)
		jsonMsg(c, "删除", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("读取响应失败:", err)
		jsonMsg(c, "删除", err)
		return
	}

	// backend-proxy 返回格式: {"success": true, "msg": "..."}
	var backendResult struct {
		Success bool   `json:"success"`
		Msg     string `json:"msg"`
	}
	if err := json.Unmarshal(body, &backendResult); err != nil {
		logger.Error("解析响应失败:", err)
		jsonMsg(c, "删除", err)
		return
	}

	if backendResult.Success {
		jsonMsg(c, "删除", nil)
	} else {
		jsonMsg(c, backendResult.Msg, fmt.Errorf(backendResult.Msg))
	}
}
