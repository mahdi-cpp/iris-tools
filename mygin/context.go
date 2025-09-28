package mygin

import (
	"net/http"
)

// Context encapsulates the request and response objects, and holds route parameters.
type Context struct {
	Writer http.ResponseWriter
	Req    *http.Request
	// Path-related fields
	Path   string
	Method string
	Params map[string]string // Key: parameter name, Value: value from request URL
	// Response status
	StatusCode int
}

// NewContext creates a new Context
func NewContext(w http.ResponseWriter, req *http.Request) *Context {
	return &Context{
		Writer: w,
		Req:    req,
		Path:   req.URL.Path,
		Method: req.Method,
	}
}

// Param returns the value of the URL parameter with the given name.
func (c *Context) Param(key string) string {
	return c.Params[key]
}
