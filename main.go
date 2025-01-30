package main

import (
	"handler/impl"
	"handler/openapi"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	// Create our server implementation
	serverImpl := impl.NewServerImpl()

	// Register the server, which is generated in server.gen.go
	openapi.RegisterHandlers(r, serverImpl)

	log.Println("Listening on :8080")
	http.ListenAndServe(":8080", r)
}
