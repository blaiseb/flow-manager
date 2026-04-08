package main

import (
	"flow-manager/database"
	"flow-manager/handlers"
	"github.com/gin-gonic/gin"
	"io"
	"log"
	"os"
)

func main() {
	// Configuration du logging dans un fichier
	logFile, err := os.OpenFile("flow-manager.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	// Rediriger les logs de Gin et les logs standards vers le fichier et la console
	gin.DefaultWriter = io.MultiWriter(logFile, os.Stdout)
	log.SetOutput(gin.DefaultWriter)
	log.SetFlags(log.LstdFlags | log.Lshortfile)


	// Initialisation de la base de données
	database.InitDatabase()

	// Création du routeur Gin
	router := gin.Default()

	// Charger les templates HTML
	router.LoadHTMLGlob("templates/*")

	// Définir la route principale qui affiche le formulaire et les résultats de recherche
	router.GET("/", handlers.ViewHandler)

	// Définir la route pour la soumission du formulaire
	router.POST("/submit", handlers.SubmitHandler)

	// Définir la route pour l'export Excel
	router.GET("/export", handlers.ExportHandler)

	// Routes for VLAN management
	router.POST("/vlan", handlers.CreateVlan)
	router.PUT("/vlan/:id", handlers.UpdateVlan)
	router.DELETE("/vlan/:id", handlers.DeleteVlan)
	router.GET("/vlan/lookup", handlers.VlanLookupHandler)
	router.POST("/vlan/import", handlers.ImportVlans)
	router.GET("/vlan/export", handlers.ExportVlans)
	// Routes for CI management
	router.POST("/ci", handlers.CreateCI)
	router.PUT("/ci/:id", handlers.UpdateCI)
	router.DELETE("/ci/:id", handlers.DeleteCI)
	router.GET("/ci/lookup", handlers.CiLookupHandler)
	router.GET("/ci/suggest", handlers.CiSuggestHandler)
	// Routes for Flow management
	router.PUT("/flow/:id", handlers.UpdateFlow)
	router.DELETE("/flow/:id", handlers.DeleteFlow)

	// Démarrer le serveur
	log.Println("Starting server on :8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
