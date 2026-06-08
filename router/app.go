package router

import (
	"ginchat/controller"

	"github.com/gin-gonic/gin"
)

func Router() *gin.Engine {
	r := gin.Default()

	// 基础路由
	r.GET("/index", controller.GetIndex)

	// 用户相关路由
	r.GET("/users", controller.GetUsers)
	r.POST("/users", controller.CreateUser)

	// 搜索相关路由
	searchGroup := r.Group("/search")
	{
		searchGroup.GET("/documents", controller.SearchDocuments)
		searchGroup.POST("/documents", controller.CreateDocument)
		searchGroup.GET("/documents/:id", controller.GetDocument)
		searchGroup.GET("/documents/:id/Extract", controller.ExtractDocument)
		searchGroup.GET("/tags", controller.GetTags)
		searchGroup.GET("/history", controller.GetSearchHistory)
		
		// 新增以下两个路由，它们必须在 searchGroup 的作用域内，且在最终闭合括号之前
		searchGroup.POST("/evaluate", controller.SubmitEvaluation)
		searchGroup.GET("/metrics", controller.GetEvaluationMetrics)
	}

	return r
}