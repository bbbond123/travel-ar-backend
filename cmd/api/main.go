package main

import (
	"net/http"

	"travel-ar-backend/internal/auth"
	"travel-ar-backend/internal/server"
)

func main() {

	auth.NewAuth()
	server := server.NewServer()

	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		panic("cannot start server")
	}

}
