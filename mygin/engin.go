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

func (group *RouterGroup) calculateAbsolutePath(relativePath string) string {
	// حذف اسلش‌های تکراری
	if relativePath == "" {
		return group.basePath
	}

	finalPath := group.basePath
	if finalPath == "/" {
		finalPath = ""
	}

	// اطمینان از اینکه relativePath با اسلش شروع می‌شود
	if relativePath[0] != '/' {
		finalPath += "/" + relativePath
	} else {
		finalPath += relativePath
	}

	// حذف اسلش انتهایی اگر وجود دارد (به جز برای مسیر ریشه)
	if len(finalPath) > 1 && finalPath[len(finalPath)-1] == '/' {
		finalPath = finalPath[:len(finalPath)-1]
	}

	return finalPath
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

func (engine *Engine) addRoute(method, path string, handlers HandlersChain) {
	if method == "" {
		panic("method must not be empty")
	}
	if len(path) < 1 || path[0] != '/' {
		panic("path must begin with '/'")
	}

	// نرمال‌سازی مسیر
	if path != "/" && path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}

	if engine.router[method] == nil {
		engine.router[method] = &node{path: "/"}
	}

	// اگر مسیر ریشه است
	if path == "/" {
		engine.router[method].handlers = handlers
		engine.router[method].fullPath = "/"
	} else {
		engine.router[method].add(path, handlers, path)
	}

	// Logging
	handlersCount := len(handlers)
	logString := formatRoutePrint(method, path, handlersCount)
	fmt.Println(logString)
}

// ServeHTTP implements the http.Handler interface.
func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {

	root := engine.router[req.Method]
	if root == nil {
		http.NotFound(w, req)
		return
	}

	handlers, params := root.find(req.URL.Path)

	if handlers != nil {
		// 1. Context را با زنجیره کامل Handlers ایجاد کنید
		c := NewContext(w, req, handlers)
		c.Params = params

		// 2. اجرای زنجیره را شروع کنید
		c.Next()

	} else {
		// مسیر پیدا نشد
		http.NotFound(w, req)
	}
}

// formatRoutePrint formats the route information for printing in the terminal.
func formatRoutePrint(method, path string, handlers int) string {
	// کد رنگ‌های ANSI
	const (
		reset   = "\033[0m"
		yellow  = "\033[33m"
		green   = "\033[32m"
		blue    = "\033[34m"
		magenta = "\033[35m"
		// استفاده از رنگ‌های روشن برای نمایش متدها
		lightBlue = "\033[94m"
	)

	// تعیین رنگ بر اساس متد HTTP
	var methodColor string
	switch method {
	case http.MethodGet:
		methodColor = blue
	case http.MethodPost:
		methodColor = green
	case http.MethodPut, http.MethodPatch:
		methodColor = yellow
	case http.MethodDelete:
		methodColor = magenta
	default:
		methodColor = reset
	}

	// 💡 رفع مشکل نمایش: استفاده از log.Printf برای زمان‌بندی و استفاده از Sprintf برای بخش متد و مسیر
	// خروجی را به این صورت تغییر می‌دهیم تا با لاگ پیش‌فرض سیستم تداخل نداشته باشد:

	return fmt.Sprintf(
		"%s%-6s%s %s%s %s(%d handlers)%s",
		methodColor,
		strings.ToUpper(method),
		reset,
		lightBlue, // رنگ مسیر
		path,
		reset,
		handlers,
		reset,
	)
	// نکته: برای جلوگیری از تداخل با log.Fatal، دیگر زمان را مستقیماً در اینجا چاپ نمی‌کنیم.
}
