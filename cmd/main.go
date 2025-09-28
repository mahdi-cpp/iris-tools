package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/mahdi-cpp/iris-tools/mygin"
)

// Logger Middleware
func Logger() mygin.HandlerFunc {
	return func(c *mygin.Context) {
		log.Printf("[%s] %s - Request received", c.Method, c.Path)
	}
}

// IndexHandler Handler for the home page
func IndexHandler(c *mygin.Context) {
	c.Writer.WriteHeader(http.StatusOK)
	c.Writer.Write([]byte("Welcome to the Home Page!"))
}

// UserProfileHandler Handler for a dynamic path
func UserProfileHandler(c *mygin.Context) {
	userID := c.Param("id")
	c.Writer.WriteHeader(http.StatusOK)
	c.Writer.Write([]byte(fmt.Sprintf("Fetching profile for User ID: %s", userID)))
}

func main() {
	r := mygin.New()

	// Global Middleware
	r.Use(Logger())

	// Static Route
	r.GET("/", IndexHandler)

	// Dynamic Route
	r.GET("/users/:id", UserProfileHandler)

	// Group with its own Middleware
	admin := r.Group("/admin")
	admin.Use(func(c *mygin.Context) {
		log.Println("Admin authentication check...")
	})

	// Route inside the group: /admin/dashboard
	admin.GET("/dashboard", func(c *mygin.Context) {
		c.Writer.Write([]byte("Welcome, Admin!"))
	})

	r.POST("/api/albums/", func(c *mygin.Context) {
		c.Writer.WriteHeader(http.StatusCreated)
		c.Writer.Write([]byte("Photos added to album!"))
	})

	// Test the problematic route from the previous query (POST with trailing slash)
	r.POST("/api/albums/photos/", func(c *mygin.Context) {
		c.Writer.WriteHeader(http.StatusCreated)
		c.Writer.Write([]byte("Photos added to album!"))
	})

	// Start the server
	fmt.Println("Server is running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
