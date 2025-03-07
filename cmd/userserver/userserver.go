package main

import (
	"github.com/gin-gonic/gin"
	"github.com/lucidstackhq/commons/env"
	"github.com/lucidstackhq/commons/logger"
	"github.com/lucidstackhq/commons/users"
	"go.uber.org/zap"
)

func main() {
	nodeID := env.GetOrDefault("NODE_ID", "node1")
	logger.Init(nodeID)
	defer func() {
		_ = zap.L().Sync()
	}()

	router := gin.Default()

	users.NewServer(
		router,
		env.GetOrDefault("USERS_RAFT_ADDRESS", "127.0.0.1:9000"),
		nodeID,
		env.GetOrDefault("BOOTSTRAP_SERVERS", "127.0.0.1:9001"),
		env.GetOrDefault("DATA_PATH", "data/node1"),
	).Load()

	err := router.Run(env.GetOrDefault("SERVER_ADDRESS", "0.0.0.0:5000"))

	if err != nil {
		zap.L().Fatal("failed to start userserver", zap.Error(err))
	}
}
