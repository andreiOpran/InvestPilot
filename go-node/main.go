package main

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Go node works"})
	})

	// endpoint that shows vpc communication
	r.POST("/simulate-investment", func(c *gin.Context) {
		// make a request to the py container using the name of the service from docker-compose
		resp, err := http.Post("http://python-engine:5000/optimize", "application/json", nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error commincating with Py node"})
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)

		// forward response to frontend
		c.Data(http.StatusOK, "application/json", body)
	})

	r.Run(":8080")
}
