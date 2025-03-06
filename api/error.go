package api

import "github.com/gin-gonic/gin"

type ErrorResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func Error(c *gin.Context, httpStatusCode int, err error) {
	ErrorMessage(c, httpStatusCode, err.Error())
}

func ErrorMessage(c *gin.Context, httpStatusCode int, err string) {
	c.AbortWithStatusJSON(httpStatusCode, &ErrorResponse{
		Success: false,
		Message: err,
	})
}
