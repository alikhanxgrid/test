// Package apiv2 provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/oapi-codegen/oapi-codegen/v2 version v2.4.1 DO NOT EDIT.
package apiv2

import (
	"github.com/gin-gonic/gin"
)

// ServerInterface represents all server handlers.
type ServerInterface interface {
	// Signal worker break status
	// (POST /break)
	SignalBreak(c *gin.Context)
	// Worker check-in endpoint
	// (POST /check-in)
	WorkerCheckIn(c *gin.Context)
	// Worker check-out endpoint
	// (POST /check-out)
	WorkerCheckOut(c *gin.Context)
	// Update task progress
	// (POST /task-progress)
	UpdateTaskProgress(c *gin.Context)
}

// ServerInterfaceWrapper converts contexts to parameters.
type ServerInterfaceWrapper struct {
	Handler            ServerInterface
	HandlerMiddlewares []MiddlewareFunc
	ErrorHandler       func(*gin.Context, error, int)
}

type MiddlewareFunc func(c *gin.Context)

// SignalBreak operation middleware
func (siw *ServerInterfaceWrapper) SignalBreak(c *gin.Context) {

	for _, middleware := range siw.HandlerMiddlewares {
		middleware(c)
		if c.IsAborted() {
			return
		}
	}

	siw.Handler.SignalBreak(c)
}

// WorkerCheckIn operation middleware
func (siw *ServerInterfaceWrapper) WorkerCheckIn(c *gin.Context) {

	for _, middleware := range siw.HandlerMiddlewares {
		middleware(c)
		if c.IsAborted() {
			return
		}
	}

	siw.Handler.WorkerCheckIn(c)
}

// WorkerCheckOut operation middleware
func (siw *ServerInterfaceWrapper) WorkerCheckOut(c *gin.Context) {

	for _, middleware := range siw.HandlerMiddlewares {
		middleware(c)
		if c.IsAborted() {
			return
		}
	}

	siw.Handler.WorkerCheckOut(c)
}

// UpdateTaskProgress operation middleware
func (siw *ServerInterfaceWrapper) UpdateTaskProgress(c *gin.Context) {

	for _, middleware := range siw.HandlerMiddlewares {
		middleware(c)
		if c.IsAborted() {
			return
		}
	}

	siw.Handler.UpdateTaskProgress(c)
}

// GinServerOptions provides options for the Gin server.
type GinServerOptions struct {
	BaseURL      string
	Middlewares  []MiddlewareFunc
	ErrorHandler func(*gin.Context, error, int)
}

// RegisterHandlers creates http.Handler with routing matching OpenAPI spec.
func RegisterHandlers(router gin.IRouter, si ServerInterface) {
	RegisterHandlersWithOptions(router, si, GinServerOptions{})
}

// RegisterHandlersWithOptions creates http.Handler with additional options
func RegisterHandlersWithOptions(router gin.IRouter, si ServerInterface, options GinServerOptions) {
	errorHandler := options.ErrorHandler
	if errorHandler == nil {
		errorHandler = func(c *gin.Context, err error, statusCode int) {
			c.JSON(statusCode, gin.H{"msg": err.Error()})
		}
	}

	wrapper := ServerInterfaceWrapper{
		Handler:            si,
		HandlerMiddlewares: options.Middlewares,
		ErrorHandler:       errorHandler,
	}

	router.POST(options.BaseURL+"/break", wrapper.SignalBreak)
	router.POST(options.BaseURL+"/check-in", wrapper.WorkerCheckIn)
	router.POST(options.BaseURL+"/check-out", wrapper.WorkerCheckOut)
	router.POST(options.BaseURL+"/task-progress", wrapper.UpdateTaskProgress)
}
