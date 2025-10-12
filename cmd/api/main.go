package main

import (
	"log"

	"anomaly-detector/internal/api"
	"anomaly-detector/internal/store"
)

// @title           API du Moteur de Détection d'Anomalies
// @version         1.0
// @description     Backend pour le projet MDA de GSIE Technology, conçu pour la détection de fraudes LBC/FT.
// @termsOfService  https://gsietechnology.com/terms
// @contact.name    Modou GUEYE
// @contact.email   modou.gueye@gsietechnology.com
// @license.name    MIT
// @license.url     https://opensource.org/licenses/MIT
// @host            localhost:8080
// @BasePath        /api/v1
// @schemes         http
// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization
// @description                 Type 'Bearer' suivi par un espace et un token JWT.
// @x-logo                      https://gsietechnology.com/logo.png
func main() {
	connStr := "postgresql://anomaly_user:passer@localhost:5432/anomaly_db?sslmode=disable"

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// 1. Initialiser la couche de persistance (le store).
	dbStore, err := store.NewStore(connStr)
	if err != nil {
		log.Fatalf("❌ Impossible d'initialiser le store: %v", err)
	}
	defer dbStore.Close()

	// 2. Initialiser le serveur API en lui passant le store.
	server := api.NewServer(dbStore)

	// 3. Démarrer le serveur.
	log.Println("🚀 Démarrage du serveur sur le port 8080")
	if err := server.Start(":8080"); err != nil {
		log.Fatalf("❌ Impossible de démarrer le serveur: %v", err)
	}
}
