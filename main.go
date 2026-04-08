package main

import (
	"flag"
	"flow-manager/config"
	"flow-manager/database"
	"flow-manager/handlers"
	"flow-manager/logger"
	"fmt"
	"io"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	// Parsing flags
	configPath := flag.String("config", "config.yaml", "Path to config file")
	debugFlag := flag.Bool("debug", false, "Force enable debug mode (overrides config)")
	portFlag := flag.Int("port", 0, "Server port (overrides config)")
	flag.Parse()

	// Load Configuration
	if err := config.LoadConfig(*configPath); err != nil {
		fmt.Printf("Warning: Could not load config file: %v. Using defaults.\n", err)
	}

	// Overrides from flags
	if *debugFlag {
		config.Global.Log.Debug = true
		config.Global.Log.Level = "debug"
	}
	if *portFlag != 0 {
		config.Global.Server.Port = *portFlag
	}

	logger.DebugMode = config.Global.Log.Debug

	// Logging initialization
	logFile, err := os.OpenFile("flow-manager.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic("Failed to open log file: " + err.Error())
	}

	logger.InitLogger(logFile, config.Global.Log.Level)

	if logger.DebugMode {
		logger.Debug("Debug mode is enabled")
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	gin.DefaultWriter = io.MultiWriter(logFile, os.Stdout)

	database.InitDatabase()

	var router *gin.Engine
	if logger.DebugMode {
		router = gin.Default()
	} else {
		router = gin.New()
		router.Use(gin.Recovery())
	}

	router.LoadHTMLGlob("templates/*")

	// Routes
	router.GET("/", handlers.ViewHandler)
	router.POST("/submit", handlers.SubmitHandler)
	router.GET("/export", handlers.ExportHandler)

	router.POST("/vlan", handlers.CreateVlan)
	router.PUT("/vlan/:id", handlers.UpdateVlan)
	router.DELETE("/vlan/:id", handlers.DeleteVlan)
	router.GET("/vlan/lookup", handlers.VlanLookupHandler)
	router.POST("/vlan/import", handlers.ImportVlans)
	router.GET("/vlan/export", handlers.ExportVlans)

	router.POST("/ci", handlers.CreateCI)
	router.PUT("/ci/:id", handlers.UpdateCI)
	router.DELETE("/ci/:id", handlers.DeleteCI)
	router.GET("/ci/lookup", handlers.CiLookupHandler)
	router.GET("/ci/suggest", handlers.CiSuggestHandler)
	router.POST("/ci/import", handlers.ImportCIs)
	router.GET("/ci/export", handlers.ExportCIs)

	router.PUT("/flow/:id", handlers.UpdateFlow)
	router.DELETE("/flow/:id", handlers.DeleteFlow)

	addr := fmt.Sprintf(":%d", config.Global.Server.Port)
	logger.Info("Starting server", "address", addr)
	if err := router.Run(addr); err != nil {
		logger.Fatal("Failed to start server", "error", err)
	}
}
