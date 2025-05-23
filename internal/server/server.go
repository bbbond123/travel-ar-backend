package server

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"travel-ar-backend/internal/database"
	"travel-ar-backend/internal/router"

	_ "github.com/joho/godotenv/autoload"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Server struct {
	port int

	db database.Service
}

func NewServer() *http.Server {

	// 3. 初始化 Gin 路由
	r := router.InitRouter()
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// 4. 端口
	port := 8080
	if p := os.Getenv("SERVER_PORT"); p != "" {
		if v, err := strconv.Atoi(p); err == nil {
			port = v
		}
	}

	// Declare Server config
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      r,
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return server
}
