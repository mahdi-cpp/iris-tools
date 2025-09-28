package mygin

import (
	"fmt"
	"net/http"
	"strings"
)

// Engine is the core struct that handles routing and implements http.Handler.
type Engine struct {
	*RouterGroup
	router map[string]*node // The Radix Tree map: Key is HTTP method (e.g., "GET")
}

// RouterGroup manages groups of routes and shared handlers (middleware).
type RouterGroup struct {
	Handlers HandlersChain
	basePath string
	engine   *Engine
}

// New creates a new Engine instance.
func New() *Engine {
	engine := &Engine{
		router: make(map[string]*node),
	}
	// Set up the default router group which points to the engine
	engine.RouterGroup = &RouterGroup{
		engine:   engine,
		basePath: "/",
	}
	return engine
}

// Group creates a new RouterGroup with a given relative path.
func (group *RouterGroup) Group(relativePath string) *RouterGroup {
	return &RouterGroup{
		engine:   group.engine,
		basePath: group.calculateAbsolutePath(relativePath),
		Handlers: group.combineHandlers(group.Handlers), // Inherit middleware
	}
}

// Use adds middleware to the group
func (group *RouterGroup) Use(middleware ...HandlerFunc) {
	group.Handlers = append(group.Handlers, middleware...)
}

// calculateAbsolutePath resolves the absolute path for a relative path.
func (group *RouterGroup) calculateAbsolutePath(relativePath string) string {
	return group.basePath + relativePath
}

// combineHandlers copies and appends handlers.
func (group *RouterGroup) combineHandlers(handlers HandlersChain) HandlersChain {
	finalSize := len(group.Handlers) + len(handlers)
	mergedHandlers := make(HandlersChain, finalSize)
	copy(mergedHandlers, group.Handlers)
	copy(mergedHandlers[len(group.Handlers):], handlers)
	return mergedHandlers
}

// GET registers a GET request handler
func (group *RouterGroup) GET(relativePath string, handlers ...HandlerFunc) {
	group.handle(http.MethodGet, relativePath, handlers)
}

// POST registers a POST request handler
func (group *RouterGroup) POST(relativePath string, handlers ...HandlerFunc) {
	group.handle(http.MethodPost, relativePath, handlers)
}

// PATCH registers a PATCH request handler
func (group *RouterGroup) PATCH(relativePath string, handlers ...HandlerFunc) {
	group.handle(http.MethodPatch, relativePath, handlers)
}

// DELETE registers a DELETE request handler
func (group *RouterGroup) DELETE(relativePath string, handlers ...HandlerFunc) {
	group.handle(http.MethodDelete, relativePath, handlers)
}

// handle registers a new request handle with the given path and method.
func (group *RouterGroup) handle(httpMethod, relativePath string, handlers HandlersChain) {
	absolutePath := group.calculateAbsolutePath(relativePath)
	handlers = group.combineHandlers(handlers)
	group.engine.addRoute(httpMethod, absolutePath, handlers)
}

// addRoute adds a route to the engine's Radix Tree.
func (engine *Engine) addRoute(method, path string, handlers HandlersChain) {
	if method == "" {
		panic("method must not be empty")
	}
	if len(path) < 1 || path[0] != '/' {
		panic("path must begin with '/'")
	}

	if engine.router[method] == nil {
		// Initialize the root node for this method
		engine.router[method] = &node{fullPath: "/"}
	}
	engine.router[method].addRoute(path, handlers)

	// =========================================================
	// 💡 به‌روزرسانی برای نمایش مسیر در ترمینال (مشابه Gin)
	// =========================================================

	// تعداد Handlerها (شامل Middlewareها و Handler نهایی)
	handlersCount := len(handlers)

	// ایجاد خروجی لاگ شده
	logString := formatRoutePrint(method, path, handlersCount)

	// چاپ در ترمینال
	fmt.Println(logString)
}

// ServeHTTP implements the http.Handler interface (required by http.ListenAndServe).
func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// 1. Find the root node for the method
	root := engine.router[req.Method]
	if root == nil {
		http.NotFound(w, req)
		return
	}

	// 2. Find the route and parameters
	handlers, params := root.find(req.URL.Path)

	if handlers != nil {
		// 3. Create Context
		c := NewContext(w, req)
		c.Params = params

		// 4. Execute Handlers (simplified: only execute the last one for now)
		if len(handlers) > 0 {
			handlers[len(handlers)-1](c)
		}

	} else {
		// 5. No route found
		http.NotFound(w, req)
	}
}

// =========================================================
// تابع کمکی برای فرمت‌دهی خروجی (به سبک Gin)
// =========================================================

// formatRoutePrint formats the route information for printing in the terminal.
func formatRoutePrint(method, path string, handlers int) string {
	// کد رنگ‌های ANSI
	const (
		reset   = "\033[0m"
		yellow  = "\033[33m"
		green   = "\033[32m"
		blue    = "\033[34m"
		magenta = "\033[35m"
	)

	// تعیین رنگ بر اساس متد HTTP
	var methodColor string
	switch method {
	case http.MethodGet:
		methodColor = blue
	case http.MethodPost:
		methodColor = green
	case http.MethodPut:
		methodColor = yellow
	case http.MethodDelete:
		methodColor = magenta
	default:
		methodColor = reset
	}

	// [Time] [Method] [Path] (Handlers Count)
	return fmt.Sprintf(
		"%s%s %-6s%s %s%s %s(%d handlers)%s",
		reset, // reset for time
		yellow,
		strings.ToUpper(method),
		reset,
		methodColor,
		path,
		reset,
		handlers,
		reset,
	)
}
