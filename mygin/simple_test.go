package mygin

import (
	"testing"
)

func TestSimpleRoute(t *testing.T) {

	router := New()

	router.POST("/api/photos", func(c *Context) {

		c.JSON(200, H{"message": "success"})
	})
	router.POST("/api/photos/ali", func(c *Context) {

		c.JSON(200, H{"message": "success"})
	})

	router.GET("/api/photos", func(c *Context) {

		c.JSON(200, H{"message": "success"})
	})
	router.GET("/api/photos/ali", func(c *Context) {

		c.JSON(200, H{"message": "success"})
	})

	// تست مستقیم درخت
	root := router.router["GET"]
	if root == nil {
		t.Fatal("GET method not registered in router")
	}

	handlers, params := root.find("/api/photos")
	if handlers == nil {
		t.Fatal("Route /api/photos not found in tree")
	}

	if len(handlers) == 0 {
		t.Fatal("No handlers found for route")
	}

	t.Logf("Route found successfully with %d handlers", len(handlers))
	t.Logf("Params: %v", params)
}
