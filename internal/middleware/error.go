package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			// Return the last error
			err := c.Errors.Last()
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
		}
	}
}
