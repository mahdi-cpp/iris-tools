package mygin

// HandlerFunc is the function signature for a handler.
type HandlerFunc func(*Context)

// HandlersChain is a slice of HandlerFunc.
type HandlersChain []HandlerFunc
