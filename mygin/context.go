package mygin

import (
	"encoding/json"
	"fmt"
	"html/template"
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
	// Response status and flow control
	StatusCode int
	index      int           // Used for managing middleware chain execution
	Handlers   HandlersChain // The chain of handlers/middlewares for this request
}

// NewContext creates a new Context.
func NewContext(w http.ResponseWriter, req *http.Request, handlers HandlersChain) *Context {
	return &Context{
		Writer:   w,
		Req:      req,
		Path:     req.URL.Path,
		Method:   req.Method,
		Handlers: handlers,
		index:    -1, // Start before the first handler
	}
}

// Param returns the value of the URL parameter with the given key (e.g., "id").
func (c *Context) Param(key string) string {
	if c.Params == nil {
		return ""
	}
	return c.Params[key]
}

// Status sets the HTTP status code for the response.
func (c *Context) Status(code int) {
	c.StatusCode = code
	c.Writer.WriteHeader(code)
}

// GetHeader returns the value of the request header key.
func (c *Context) GetHeader(key string) string {
	return c.Req.Header.Get(key)
}

// --- توابع کنترل جریان (Middleware Flow Control) ---

// Next should be called in a middleware to execute the pending handlers.
func (c *Context) Next() {
	c.index++
	for c.index < len(c.Handlers) {
		c.Handlers[c.index](c)
		c.index++
	}
}

// Abort prevents pending handlers from being called.
func (c *Context) Abort() {
	c.index = len(c.Handlers)
}

// --- توابع پاسخ‌دهی (Response Helpers) ---

// JSON sends a JSON response.
func (c *Context) JSON(code int, obj interface{}) {
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Status(code)

	// Attempt to encode JSON
	if err := json.NewEncoder(c.Writer).Encode(obj); err != nil {
		http.Error(c.Writer, "JSON encoding error: "+err.Error(), http.StatusInternalServerError)
	}
}

// HTML sends an HTML response by executing a template.
// NOTE: This simple version relies on finding the template by name (file path).
func (c *Context) HTML(code int, name string, data interface{}) {
	c.Writer.Header().Set("Content-Type", "text/html")
	c.Status(code)

	// Load and parse the template file
	t, err := template.ParseFiles(name)
	if err != nil {
		http.Error(c.Writer, "Template loading error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Execute the template with provided data
	if err := t.Execute(c.Writer, data); err != nil {
		http.Error(c.Writer, "Template execution error: "+err.Error(), http.StatusInternalServerError)
	}
}

// String sends a plain text response.
func (c *Context) String(code int, format string, values ...interface{}) {
	c.Writer.Header().Set("Content-Type", "text/plain")
	c.Status(code)

	// Use fmt.Sprintf for string formatting
	c.Writer.Write([]byte(fmt.Sprintf(format, values...)))
}

// Data sends raw byte data response.
func (c *Context) Data(code int, contentType string, data []byte) {
	c.Writer.Header().Set("Content-Type", contentType)
	c.Status(code)
	c.Writer.Write(data)
}
