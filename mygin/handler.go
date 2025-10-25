package mygin

// H is a shortcut for map[string]interface{}, similar to gin.H
type H map[string]interface{}

// HandlerFunc is the function signature for a request handler.
type HandlerFunc func(*Context)

// HandlersChain is a slice of HandlerFunc (used for middlewares and the final handler).
type HandlersChain []HandlerFunc
