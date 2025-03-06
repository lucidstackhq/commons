package api

import (
	"github.com/gin-gonic/gin"
)

type SuccessResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func Success(c *gin.Context, httpStatusCode int, message string) {
	c.JSON(httpStatusCode, &SuccessResponse{
		Success: true,
		Message: message,
	})
}
