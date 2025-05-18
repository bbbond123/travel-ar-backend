package router

import (
	"travel-ar-backend/internal/controller"
	"travel-ar-backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

// ArticleRouter 文章路由模块
type ArticleRouter struct{}

// Register 注册文章路由
func (ArticleRouter) Register(api *gin.RouterGroup) {
	article := api.Group("/articles")
	api.GET("/articles/:article_id", controller.GetArticle)
	api.POST("/articles/list", controller.ListArticles)

	aritcleAuth := article.Group("/articles")
	aritcleAuth.Use(middleware.JWTAuth())
	{
		article.POST("", controller.CreateArticle)
		article.PUT("", controller.UpdateArticle)
		article.DELETE(":article_id", controller.DeleteArticle)
	}
}

func init() {
	Register(ArticleRouter{})
}
