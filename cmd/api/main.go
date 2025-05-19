package main

import (
	"net/http"

	"travel-ar-backend/internal/auth"
	"travel-ar-backend/internal/server"
	"travel-ar-backend/pkg/database"
)

func main() {
	database.ConnectDatabase()
	gormDB := database.GetDB()
	auth.NewAuth()
	server := server.NewServer(gormDB)

	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		panic("cannot start server")
	}

}
