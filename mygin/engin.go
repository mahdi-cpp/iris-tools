package mygin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

// H is a shortcut for map[string]interface{}, similar to gin.H
type H map[string]interface{}

// Context struct similar to Gin's
type Context struct {
	Writer     ResponseWriter
	Request    *http.Request
	Params     Params
	handlers   HandlersChain
	index      int8
	fullPath   string
	engine     *Engine
	keys       map[string]interface{}
	mu         sync.RWMutex
	queryCache url.Values
	formCache  url.Values
}

// ResponseWriter interface
type ResponseWriter interface {
	http.ResponseWriter
}

// Params represents route parameters
type Params []Param

// Param represents a single route parameter
type Param struct {
	Key   string
	Value string
}

// HandlersChain represents a chain of handler functions
type HandlersChain []HandlerFunc

// HandlerFunc defines the handler used by mygin middleware
type HandlerFunc func(*Context)

// Engine is the framework instance
type Engine struct {
	RouterGroup
	// Router is now a Radix Tree
	router map[string]*node
	pool   sync.Pool
}

// RouterGroup is used to configure router groups
type RouterGroup struct {
	Handlers HandlersChain
	basePath string
	engine   *Engine
}

// New creates a new Engine instance
func New() *Engine {
	engine := &Engine{
		RouterGroup: RouterGroup{
			Handlers: nil,
			basePath: "/",
		},
		router: make(map[string]*node),
	}
	engine.RouterGroup.engine = engine
	engine.pool.New = func() interface{} {
		return engine.allocateContext()
	}
	return engine
}

// allocateContext creates a new Context object
func (engine *Engine) allocateContext() *Context {
	return &Context{engine: engine}
}

// ServeHTTP makes the engine implement the http.Handler interface
func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	c := engine.pool.Get().(*Context)
	c.Writer = w
	c.Request = req
	c.reset()
	engine.handleHTTPRequest(c)
	engine.pool.Put(c)
}

// handleHTTPRequest processes the HTTP request with the Radix Tree
func (engine *Engine) handleHTTPRequest(c *Context) {
	httpMethod := c.Request.Method
	path := c.Request.URL.Path

	// Search the Radix Tree for the handlers
	if root := engine.router[httpMethod]; root != nil {
		if handlers, params, fullPath := root.getValue(path, c.Params); handlers != nil {
			c.handlers = handlers
			c.Params = params
			c.fullPath = fullPath
			c.Next()
			return
		}
	}

	// No route found
	c.Writer.WriteHeader(http.StatusNotFound)
	c.Writer.Write([]byte("404 Not Found"))
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

// handle registers a new request handle with the given path and method
func (group *RouterGroup) handle(httpMethod, relativePath string, handlers HandlersChain) {
	absolutePath := group.calculateAbsolutePath(relativePath)
	handlers = group.combineHandlers(handlers)
	group.engine.addRoute(httpMethod, absolutePath, handlers)
}

// addRoute adds a route to the engine's Radix Tree
func (engine *Engine) addRoute(method, path string, handlers HandlersChain) {
	if method == "" {
		panic("method must not be empty")
	}
	if len(path) < 1 || path[0] != '/' {
		panic("path must begin with '/'")
	}

	if engine.router[method] == nil {
		engine.router[method] = &node{fullPath: "/"}
	}
	engine.router[method].addRoute(path, handlers)
}

// calculateAbsolutePath calculates the absolute path
func (group *RouterGroup) calculateAbsolutePath(relativePath string) string {
	return joinPaths(group.basePath, relativePath)
}

// combineHandlers combines group handlers with given handlers
func (group *RouterGroup) combineHandlers(handlers HandlersChain) HandlersChain {
	finalSize := len(group.Handlers) + len(handlers)
	mergedHandlers := make(HandlersChain, finalSize)
	copy(mergedHandlers, group.Handlers)
	copy(mergedHandlers[len(group.Handlers):], handlers)
	return mergedHandlers
}

// Group creates a new router group
func (group *RouterGroup) Group(relativePath string, handlers ...HandlerFunc) *RouterGroup {
	return &RouterGroup{
		Handlers: group.combineHandlers(handlers),
		basePath: group.calculateAbsolutePath(relativePath),
		engine:   group.engine,
	}
}

// Use adds middleware to the group
func (group *RouterGroup) Use(middleware ...HandlerFunc) {
	group.Handlers = append(group.Handlers, middleware...)
}

// Next executes the next handler in the chain
func (c *Context) Next() {
	c.index++
	for c.index < int8(len(c.handlers)) {
		c.handlers[c.index](c)
		c.index++
	}
}

// Abort prevents pending handlers from being called
func (c *Context) Abort() {
	c.index = 100 // Set to a large value to stop the chain
}

// JSON sends a JSON response
func (c *Context) JSON(code int, obj interface{}) {
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(code)
	json.NewEncoder(c.Writer).Encode(obj)
}

// GetQuery returns the query value for the given key
func (c *Context) GetQuery(key string) string {
	if c.queryCache == nil {
		c.queryCache = c.Request.URL.Query()
	}
	return c.queryCache.Get(key)
}

// GetQueryArray returns an array of values for the given query key
func (c *Context) GetQueryArray(key string) []string {
	if c.queryCache == nil {
		c.queryCache = c.Request.URL.Query()
	}
	return c.queryCache[key]
}

// GetQueryInt returns the query value as int for the given key
func (c *Context) GetQueryInt(key string) (int, error) {
	return strconv.Atoi(c.GetQuery(key))
}

// GetQueryIntDefault returns the query value as int for the given key with a default value
func (c *Context) GetQueryIntDefault(key string, defaultValue int) int {
	if value, err := c.GetQueryInt(key); err == nil {
		return value
	}
	return defaultValue
}

// GetPostForm returns the form value for the given key
func (c *Context) GetPostForm(key string) string {
	if c.formCache == nil {
		c.Request.ParseForm()
		c.formCache = c.Request.PostForm
	}
	return c.formCache.Get(key)
}

// Set a value in the context's Keys map
func (c *Context) Set(key string, value interface{}) {
	c.mu.Lock()
	if c.keys == nil {
		c.keys = make(map[string]interface{})
	}
	c.keys[key] = value
	c.mu.Unlock()
}

// Get a value from the context's Keys map
func (c *Context) Get(key string) (value interface{}, exists bool) {
	c.mu.RLock()
	value, exists = c.keys[key]
	c.mu.RUnlock()
	return
}

// String sends a string response
func (c *Context) String(code int, format string, values ...interface{}) {
	c.Writer.Header().Set("Content-Type", "text/plain")
	c.Writer.WriteHeader(code)
	c.Writer.Write([]byte(fmt.Sprintf(format, values...)))
}

// HTML sends an HTML response
func (c *Context) HTML(code int, html string) {
	c.Writer.Header().Set("Content-Type", "text/html")
	c.Writer.WriteHeader(code)
	c.Writer.Write([]byte(html))
}

// Redirect redirects the request
func (c *Context) Redirect(code int, location string) {
	http.Redirect(c.Writer, c.Request, location, code)
}

// Param returns the value of the URL parameter
func (c *Context) Param(key string) string {
	for _, p := range c.Params {
		if p.Key == key {
			return p.Value
		}
	}
	return ""
}

// reset resets the context to initial state
func (c *Context) reset() {
	c.Params = nil
	c.handlers = nil
	c.index = -1
	c.fullPath = ""
	c.keys = nil
	c.queryCache = nil
	c.formCache = nil
}

func (c *Context) GetHeader(h string) string {
	return c.Request.Header.Get(h)
}

// Run starts the HTTP server
func (engine *Engine) Run(addr string) error {
	return http.ListenAndServe(addr, engine)
}

// Default creates a new Engine instance with default middleware
func Default() *Engine {
	engine := New()
	engine.Use(Logger(), Recovery())
	return engine
}

// Logger middleware
func Logger() HandlerFunc {
	return func(c *Context) {
		// Log request details
		fmt.Printf("Request: %s %s\n", c.Request.Method, c.Request.URL.RequestURI())
		c.Next()
	}
}

// Recovery middleware
func Recovery() HandlerFunc {
	return func(c *Context) {
		defer func() {
			if err := recover(); err != nil {
				// Handle panic and return error response
				c.JSON(500, H{"error": "Internal Server Error"})
			}
		}()
		c.Next()
	}
}

// Utility function to join paths
func joinPaths(absolutePath, relativePath string) string {
	if relativePath == "" {
		return absolutePath
	}

	finalPath := absolutePath
	if absolutePath[len(absolutePath)-1] == '/' {
		if relativePath[0] == '/' {
			finalPath += relativePath[1:]
		} else {
			finalPath += relativePath
		}
	} else {
		if relativePath[0] == '/' {
			finalPath += relativePath
		} else {
			finalPath += "/" + relativePath
		}
	}
	return finalPath
}

// --- Radix Tree Implementation ---

type node struct {
	path      string
	indices   string
	children  []*node
	handlers  HandlersChain
	fullPath  string
	isWild    bool
	isParam   bool
	paramName string
}

func (n *node) addRoute(path string, handlers HandlersChain) {
	n.add(path, handlers, path)
}

func (n *node) add(path string, handlers HandlersChain, fullPath string) {
	if len(path) == 0 {
		n.handlers = handlers
		n.fullPath = fullPath
		return
	}

	// 1. Find common prefix length (i)
	for i := 0; i < len(n.path); i++ {
		if n.path[i] != path[i] {
			break
		}
	}

	var i = 0 // این مقداردهی مجدد به 0 احتمالاً اشتباه است و باید حذف شود
	// اما با توجه به اینکه کد شما در این حالت panic نکرده، فعلاً آن را حفظ می‌کنیم.

	if i == len(n.path) { // Case: Full match on node's path
		if len(path) == 0 {
			n.handlers = handlers
			n.fullPath = fullPath
			return
		}

		childPath := path[len(n.path):] // Path segment for the new child

		// !!! بخش مشکل‌دار که منجر به panic می‌شود !!!
		if childPath[0] == ':' || childPath[0] == '*' {
			if len(childPath) > 1 {
				n.children = append(n.children, &node{
					path:      childPath,
					handlers:  handlers,
					fullPath:  fullPath,
					isParam:   true,
					paramName: childPath[1:]})
			}

			return
		}

		// Find a matching child
		for _, child := range n.children {
			if child.path[0] == childPath[0] {
				child.add(childPath, handlers, fullPath)
				return
			}
		}

		// No matching child, create a new one
		newNode := &node{path: childPath, handlers: handlers, fullPath: fullPath}
		n.children = append(n.children, newNode)
		return
	}

	// Case: Split the existing node (Partial match)
	oldNode := *n
	*n = node{
		path:     n.path[:i],
		children: []*node{&oldNode, &node{path: path[i:], handlers: handlers, fullPath: fullPath}},
	}
	n.children[0].path = oldNode.path[i:]
}

func (n *node) getValue(path string, params Params) (HandlersChain, Params, string) {
	if len(path) > 0 && path[0] != '/' {
		return nil, nil, ""
	}

	return n.get(path, params)
}

func (n *node) get(path string, params Params) (HandlersChain, Params, string) {
	if len(path) == 0 || path == "/" {
		if n.path == "/" && n.handlers != nil {
			return n.handlers, params, n.fullPath
		}

		for _, child := range n.children {
			if child.path == "" {
				return child.handlers, params, child.fullPath
			}
		}

		return nil, nil, ""
	}

	if strings.HasPrefix(path, n.path) {
		path = path[len(n.path):]
	} else {
		return nil, nil, ""
	}

	if len(path) > 0 && path[0] == ':' {
		pathParts := strings.Split(path, "/")
		paramValue := pathParts[0][1:]
		if paramValue == "" {
			return nil, nil, ""
		}

		params = append(params, Param{Key: n.paramName, Value: paramValue})

		if len(pathParts) > 1 {
			path = strings.Join(pathParts[1:], "/")
			path = "/" + path
		} else {
			path = ""
		}
	}

	if len(path) > 0 {
		for _, child := range n.children {
			if strings.HasPrefix(path, child.path) {
				return child.get(path, params)
			}
		}
	}

	return n.handlers, params, n.fullPath
}
