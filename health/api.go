package health

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

type CheckApi struct {
	router *gin.Engine
}

func NewCheckApi(router *gin.Engine) *CheckApi {
	return &CheckApi{
		router: router,
	}
}

func (a *CheckApi) Register() {

	a.router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, &ShallowCheckResponse{Status: "ok"})
	})
}
