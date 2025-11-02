package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// ServerSite VPN服务器站点模型
type ServerSite struct {
	Id          int    `gorm:"primaryKey;autoIncrement" json:"id"`
	CDNAddress  string `gorm:"uniqueIndex;not null" json:"cdnAddress"` // CDN地址
	CountryCode string `gorm:"index;not null" json:"countryCode"`      // 国家代码（如：CN, US, JP）
	IsAvailable bool   `gorm:"default:true" json:"isAvailable"`        // 是否可用
	Latency     int64  `gorm:"default:0" json:"latency"`               // 延迟时长（毫秒）
	CreatedAt   int64  `gorm:"autoCreateTime" json:"createdAt"`
	Remark      string `json:"remark"` // 备注
}

// TableName 指定表名
func (ServerSite) TableName() string {
	return "server_sites"
}

// 服务器站点请求结构
type CreateServerSiteRequest struct {
	CDNAddress  string `json:"cdnAddress" binding:"required"`  // CDN地址（必填）
	CountryCode string `json:"countryCode" binding:"required"` // 国家代码（必填）
	IsAvailable bool   `json:"isAvailable"`                    // 是否可用（可选，默认true）
	Latency     int64  `json:"latency"`                        // 延迟时长（可选，默认0）
	Remark      string `json:"remark"`                         // 备注（可选）
}

type UpdateServerSiteRequest struct {
	CDNAddress  *string `json:"cdnAddress"`  // CDN地址（可选）
	CountryCode *string `json:"countryCode"` // 国家代码（可选）
	IsAvailable *bool   `json:"isAvailable"` // 是否可用（可选）
	Latency     *int64  `json:"latency"`     // 延迟时长（可选）
	Remark      *string `json:"remark"`      // 备注（可选）
}

// 初始化服务器站点数据库表
func initServerSiteDB() error {
	// 检查数据库文件是否存在
	if _, err := os.Stat(config.XUIDBPath); os.IsNotExist(err) {
		return fmt.Errorf("数据库文件不存在: %s", config.XUIDBPath)
	}

	// 连接数据库
	db, err := gorm.Open(sqlite.Open(config.XUIDBPath), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("连接数据库失败: %v", err)
	}

	// 自动迁移创建表
	err = db.AutoMigrate(&ServerSite{})
	if err != nil {
		return fmt.Errorf("创建 server_sites 表失败: %v", err)
	}

	log.Println("✅ 服务器站点数据库表初始化成功")
	return nil
}

// 自动插入当前服务器到数据库
func autoInsertCurrentServer() error {
	// 检查数据库文件是否存在
	if _, err := os.Stat(config.XUIDBPath); os.IsNotExist(err) {
		return fmt.Errorf("数据库文件不存在: %s", config.XUIDBPath)
	}

	// 连接数据库
	db, err := gorm.Open(sqlite.Open(config.XUIDBPath), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("连接数据库失败: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	// 确定CDN地址
	cdnAddress := config.ServerCDNAddr

	// 确定国家代码
	countryCode := config.ServerCountry

	// 检查服务器是否已存在
	var existingServer ServerSite
	result := db.Where("cdn_address = ?", cdnAddress).First(&existingServer)
	if result.Error == nil {
		// 服务器已存在，更新信息（确保国家代码等是最新的）
		updates := map[string]interface{}{
			"country_code": countryCode,
			"is_available": true,
		}
		// 如果备注为空，添加默认备注
		if existingServer.Remark == "" {
			updates["remark"] = "MainServer"
		}

		err = db.Model(&existingServer).Updates(updates).Error
		if err != nil {
			return fmt.Errorf("更新当前服务器信息失败: %v", err)
		}
		log.Printf("✅ 当前服务器已存在，已更新: %s (国家代码: %s)", cdnAddress, countryCode)
		return nil
	}

	// 服务器不存在，创建新记录
	now := time.Now().Unix()
	newServer := ServerSite{
		CDNAddress:  cdnAddress,
		CountryCode: countryCode,
		IsAvailable: true,
		Latency:     0, // 初始延迟为0，后续可以通过ping或其他方式更新
		Remark:      "MainServer",
		CreatedAt:   now,
	}

	err = db.Create(&newServer).Error
	if err != nil {
		return fmt.Errorf("创建当前服务器记录失败: %v", err)
	}

	log.Printf("✅ 当前服务器已自动插入数据库: %s (国家代码: %s)", cdnAddress, countryCode)
	return nil
}

// 获取服务器列表
func getServerSitesList(c *gin.Context) {
	// 获取查询参数
	countryCode := c.Query("countryCode")           // 可选：按国家代码筛选
	availableOnly := c.Query("available") == "true" // 可选：只返回可用的服务器

	// 检查数据库文件是否存在
	if _, err := os.Stat(config.XUIDBPath); os.IsNotExist(err) {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Msg:     fmt.Sprintf("数据库文件不存在: %s", config.XUIDBPath),
		})
		return
	}

	// 连接数据库
	db, err := gorm.Open(sqlite.Open(config.XUIDBPath), &gorm.Config{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Msg:     fmt.Sprintf("连接数据库失败: %v", err),
		})
		return
	}
	defer func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	// 构建查询
	query := db.Model(&ServerSite{})

	// 按国家代码筛选
	if countryCode != "" {
		query = query.Where("country_code = ?", countryCode)
	}

	// 只返回可用的服务器
	if availableOnly {
		query = query.Where("is_available = ?", true)
	}

	// 查询数据
	var servers []ServerSite
	err = query.Order("latency ASC, created_at DESC").Find(&servers).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Msg:     fmt.Sprintf("查询服务器列表失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Msg:     "获取服务器列表成功",
		Data:    servers,
	})
}

// 创建服务器站点
func createServerSite(c *gin.Context) {
	var req CreateServerSiteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Msg:     "请求参数错误: " + err.Error(),
		})
		return
	}

	// 检查数据库文件是否存在
	if _, err := os.Stat(config.XUIDBPath); os.IsNotExist(err) {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Msg:     fmt.Sprintf("数据库文件不存在: %s", config.XUIDBPath),
		})
		return
	}

	// 连接数据库
	db, err := gorm.Open(sqlite.Open(config.XUIDBPath), &gorm.Config{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Msg:     fmt.Sprintf("连接数据库失败: %v", err),
		})
		return
	}
	defer func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	// 检查CDN地址是否已存在
	var existingServer ServerSite
	result := db.Where("cdn_address = ?", req.CDNAddress).First(&existingServer)
	if result.Error == nil {
		c.JSON(http.StatusConflict, Response{
			Success: false,
			Msg:     "CDN地址已存在",
		})
		return
	}

	// 创建新服务器
	now := time.Now().Unix()
	newServer := ServerSite{
		CDNAddress:  req.CDNAddress,
		CountryCode: req.CountryCode,
		IsAvailable: req.IsAvailable,
		Latency:     req.Latency,
		Remark:      req.Remark,
		CreatedAt:   now,
	}

	// 设置默认值
	if req.Latency < 0 {
		newServer.Latency = 0
	}

	err = db.Create(&newServer).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Msg:     fmt.Sprintf("创建服务器失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Msg:     "创建服务器成功",
		Data:    newServer,
	})
}

// 更新服务器站点
func updateServerSite(c *gin.Context) {
	id := c.Param("id")

	var req UpdateServerSiteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Msg:     "请求参数错误: " + err.Error(),
		})
		return
	}

	// 检查数据库文件是否存在
	if _, err := os.Stat(config.XUIDBPath); os.IsNotExist(err) {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Msg:     fmt.Sprintf("数据库文件不存在: %s", config.XUIDBPath),
		})
		return
	}

	// 连接数据库
	db, err := gorm.Open(sqlite.Open(config.XUIDBPath), &gorm.Config{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Msg:     fmt.Sprintf("连接数据库失败: %v", err),
		})
		return
	}
	defer func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	// 检查服务器是否存在
	var server ServerSite
	err = db.First(&server, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, Response{
				Success: false,
				Msg:     "服务器不存在",
			})
		} else {
			c.JSON(http.StatusInternalServerError, Response{
				Success: false,
				Msg:     fmt.Sprintf("查询服务器失败: %v", err),
			})
		}
		return
	}

	// 如果更新CDN地址，检查是否与其他服务器冲突
	if req.CDNAddress != nil && *req.CDNAddress != server.CDNAddress {
		var existingServer ServerSite
		result := db.Where("cdn_address = ? AND id != ?", *req.CDNAddress, id).First(&existingServer)
		if result.Error == nil {
			c.JSON(http.StatusConflict, Response{
				Success: false,
				Msg:     "CDN地址已被其他服务器使用",
			})
			return
		}
	}

	// 构建更新数据
	updates := make(map[string]interface{})
	if req.CDNAddress != nil {
		updates["cdn_address"] = *req.CDNAddress
	}
	if req.CountryCode != nil {
		updates["country_code"] = *req.CountryCode
	}
	if req.IsAvailable != nil {
		updates["is_available"] = *req.IsAvailable
	}
	if req.Latency != nil {
		updates["latency"] = *req.Latency
	}
	if req.Remark != nil {
		updates["remark"] = *req.Remark
	}
	updates["updated_at"] = time.Now().Unix()

	// 更新服务器
	err = db.Model(&server).Updates(updates).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Msg:     fmt.Sprintf("更新服务器失败: %v", err),
		})
		return
	}

	// 重新查询更新后的数据
	db.First(&server, id)

	c.JSON(http.StatusOK, Response{
		Success: true,
		Msg:     "更新服务器成功",
		Data:    server,
	})
}

// 删除服务器站点
func deleteServerSite(c *gin.Context) {
	id := c.Param("id")

	// 检查数据库文件是否存在
	if _, err := os.Stat(config.XUIDBPath); os.IsNotExist(err) {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Msg:     fmt.Sprintf("数据库文件不存在: %s", config.XUIDBPath),
		})
		return
	}

	// 连接数据库
	db, err := gorm.Open(sqlite.Open(config.XUIDBPath), &gorm.Config{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Msg:     fmt.Sprintf("连接数据库失败: %v", err),
		})
		return
	}
	defer func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	// 检查服务器是否存在
	var server ServerSite
	err = db.First(&server, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, Response{
				Success: false,
				Msg:     "服务器不存在",
			})
		} else {
			c.JSON(http.StatusInternalServerError, Response{
				Success: false,
				Msg:     fmt.Sprintf("查询服务器失败: %v", err),
			})
		}
		return
	}

	// 删除服务器
	err = db.Delete(&server).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Msg:     fmt.Sprintf("删除服务器失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Msg:     "删除服务器成功",
	})
}

// createServerSiteTest 测试接口：快速创建服务器站点（通过浏览器访问）
// 访问示例: http://localhost:8080/test/create-server?cdnAddress=cdn.example.com&countryCode=US&latency=100
func createServerSiteTest(c *gin.Context) {
	// 从URL查询参数获取数据
	cdnAddress := c.Query("cdnAddress")
	countryCode := c.Query("countryCode")
	latencyStr := c.Query("latency")
	isAvailableStr := c.Query("isAvailable")
	remark := c.Query("remark")

	// 参数验证
	if cdnAddress == "" {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Msg:     "参数错误: cdnAddress 不能为空",
		})
		return
	}

	if countryCode == "" {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Msg:     "参数错误: countryCode 不能为空",
		})
		return
	}

	// 解析延迟时长
	var latency int64 = 0
	if latencyStr != "" {
		var err error
		latency, err = strconv.ParseInt(latencyStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, Response{
				Success: false,
				Msg:     fmt.Sprintf("参数错误: latency 必须是有效的数字，错误: %v", err),
			})
			return
		}
	}

	// 解析是否可用
	isAvailable := true
	if isAvailableStr != "" {
		isAvailable = isAvailableStr == "true" || isAvailableStr == "1"
	}

	// 检查数据库文件是否存在
	if _, err := os.Stat(config.XUIDBPath); os.IsNotExist(err) {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Msg:     fmt.Sprintf("数据库文件不存在: %s", config.XUIDBPath),
		})
		return
	}

	// 连接数据库
	db, err := gorm.Open(sqlite.Open(config.XUIDBPath), &gorm.Config{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Msg:     fmt.Sprintf("连接数据库失败: %v", err),
		})
		return
	}
	defer func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	// 检查CDN地址是否已存在
	var existingServer ServerSite
	result := db.Where("cdn_address = ?", cdnAddress).First(&existingServer)
	if result.Error == nil {
		// 已存在，更新
		now := time.Now().Unix()
		updates := map[string]interface{}{
			"country_code": countryCode,
			"is_available": isAvailable,
			"latency":      latency,
			"updated_at":   now,
		}
		if remark != "" {
			updates["remark"] = remark
		}

		err = db.Model(&existingServer).Updates(updates).Error
		if err != nil {
			c.JSON(http.StatusInternalServerError, Response{
				Success: false,
				Msg:     fmt.Sprintf("更新服务器失败: %v", err),
			})
			return
		}

		db.First(&existingServer, existingServer.Id)
		c.JSON(http.StatusOK, Response{
			Success: true,
			Msg:     fmt.Sprintf("服务器已更新: %s", cdnAddress),
			Data:    existingServer,
		})
		return
	}

	// 创建新服务器
	now := time.Now().Unix()
	newServer := ServerSite{
		CDNAddress:  cdnAddress,
		CountryCode: countryCode,
		IsAvailable: isAvailable,
		Latency:     latency,
		Remark:      remark,
		CreatedAt:   now,
	}

	err = db.Create(&newServer).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Msg:     fmt.Sprintf("创建服务器失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Msg:     fmt.Sprintf("服务器已创建: %s", cdnAddress),
		Data:    newServer,
	})
}
