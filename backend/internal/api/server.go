package api

import (
	"os"
	"strings"

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

	ginMode := strings.ToLower(os.Getenv("GIN_MODE"))
	if ginMode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())

	// En mode debug / dev, conserver le logger Gin. En mode release, supprimer la contention stdout.
	if ginMode != "release" {
		router.Use(gin.Logger())
	}

	allHandlers := routes.NewAllHandlers(store)
	routes.SetupRoutes(router, allHandlers, store)
	server.router = router
	return server
}

func (server *Server) Start(address string) error {
	return server.router.Run(address)
}
