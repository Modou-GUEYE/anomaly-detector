package api

import (
	"anomaly-detector/internal/api/routes"
	"anomaly-detector/internal/store"

	_ "anomaly-detector/docs"

	"github.com/gin-gonic/gin"
)

type Server struct {
	store  *store.Store
	router *gin.Engine
}

func NewServer(store *store.Store) *Server {
	server := &Server{store: store}
	router := gin.Default()
	allHandlers := routes.NewAllHandlers(store)
	routes.SetupRoutes(router, allHandlers, store)
	server.router = router
	return server
}

func (server *Server) Start(address string) error {
	return server.router.Run(address)
}
