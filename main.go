package main

import (
	"flag"
	"flow-manager/database"
	"flow-manager/handlers"
	"flow-manager/logger"
	"io"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	// Parsing flags
	debug := flag.Bool("debug", false, "Enable debug logging and development mode")
	flag.Parse()

	// Setting global debug mode
	logger.DebugMode = *debug

	// Configuration du logging dans un fichier
	logFile, err := os.OpenFile("flow-manager.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		// On ne peut pas encore utiliser le logger personnalisé car il n'est pas initialisé
		panic("Failed to open log file: " + err.Error())
	}

	// Initialisation du logger personnalisé (log/slog)
	logger.InitLogger(logFile)

	if logger.DebugMode {
		logger.Debug("Debug mode is enabled")
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Rediriger les logs de Gin vers notre multi-writer
	gin.DefaultWriter = io.MultiWriter(logFile, os.Stdout)

	// Initialisation de la base de données
	database.InitDatabase()

	// Création du routeur Gin
	var router *gin.Engine
	if logger.DebugMode {
		router = gin.Default() // Includes Logger and Recovery
	} else {
		router = gin.New()
		router.Use(gin.Recovery()) // Still recover from panics, but no access logs
	}

	// Charger les templates HTML
	router.LoadHTMLGlob("templates/*")

	// Routes
	router.GET("/", handlers.ViewHandler)
	router.POST("/submit", handlers.SubmitHandler)
	router.GET("/export", handlers.ExportHandler)

	// VLAN management
	router.POST("/vlan", handlers.CreateVlan)
	router.PUT("/vlan/:id", handlers.UpdateVlan)
	router.DELETE("/vlan/:id", handlers.DeleteVlan)
	router.GET("/vlan/lookup", handlers.VlanLookupHandler)
	router.POST("/vlan/import", handlers.ImportVlans)
	router.GET("/vlan/export", handlers.ExportVlans)

	// CI management
	router.POST("/ci", handlers.CreateCI)
	router.PUT("/ci/:id", handlers.UpdateCI)
	router.DELETE("/ci/:id", handlers.DeleteCI)
	router.GET("/ci/lookup", handlers.CiLookupHandler)
	router.GET("/ci/suggest", handlers.CiSuggestHandler)

	// Flow management
	router.PUT("/flow/:id", handlers.UpdateFlow)
	router.DELETE("/flow/:id", handlers.DeleteFlow)

	// Démarrer le serveur
	logger.Info("Starting server on :8080")
	if err := router.Run(":8080"); err != nil {
		logger.Fatal("Failed to start server", "error", err)
	}
}
