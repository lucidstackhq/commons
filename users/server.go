package users

import (
	"github.com/gin-gonic/gin"
	"github.com/lucidstackhq/commons/api"
	"go.uber.org/zap"
	"net/http"
)

type Server struct {
	router           *gin.Engine
	raftAddress      string
	nodeID           string
	bootstrapServers string
	dataPath         string
}

func NewServer(router *gin.Engine, raftAddress string, nodeID string, bootstrapServers string, dataPath string) *Server {
	return &Server{router: router, raftAddress: raftAddress, nodeID: nodeID, bootstrapServers: bootstrapServers, dataPath: dataPath}
}

func (s *Server) Load() {
	store := NewStore(s.dataPath)
	err := store.setupRaft(s.raftAddress, s.bootstrapServers)
	if err != nil {
		zap.L().Error("failed to set up raft: %v", zap.Error(err))
	}

	userRouter := s.router.Group("/api/v1/users")

	userRouter.GET("/store", func(context *gin.Context) {
		context.JSON(http.StatusOK, store.GetInfo())
	})

	userRouter.POST("/", func(context *gin.Context) {
		username, ok := context.GetQuery("username")
		if !ok {
			api.ErrorMessage(context, http.StatusBadRequest, "username is required")
			return
		}
		password, ok := context.GetQuery("password")
		if !ok {
			api.ErrorMessage(context, http.StatusBadRequest, "password is required")
			return
		}

		err := store.Set(username, password)
		if err != nil {
			api.Error(context, http.StatusInternalServerError, err)
			return
		}

		api.Success(context, http.StatusCreated, "user created successfully")
	})

	userRouter.GET("/", func(context *gin.Context) {
		username, ok := context.GetQuery("username")
		if !ok {
			api.ErrorMessage(context, http.StatusBadRequest, "username is required")
			return
		}

		_, err = store.Get(username)

		if err != nil {
			api.Error(context, http.StatusInternalServerError, err)
			return
		}

		context.JSON(http.StatusOK, gin.H{"username": username})
	})
}
