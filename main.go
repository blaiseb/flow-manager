package main

import (
	"flag"
	"flow-manager/auth"
	"flow-manager/config"
	"flow-manager/database"
	"flow-manager/handlers"
	"flow-manager/logger"
	"flow-manager/models"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
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

	if *debugFlag {
		config.Global.Log.Debug = true
		config.Global.Log.Level = "debug"
	}
	if *portFlag != 0 {
		config.Global.Server.Port = *portFlag
	}

	logger.DebugMode = config.Global.Log.Debug

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
	handlers.InitOIDC()

	// Ensure admin user has hashed password
	var admin models.User
	err = database.DB.Where("username = ?", "admin").First(&admin).Error
	if err == nil {
		// If password is still plain "admin", hash it
		if admin.Password == "admin" || !strings.HasPrefix(admin.Password, "$2a$") {
			hashed, _ := auth.HashPassword("admin")
			admin.Password = hashed
			database.DB.Save(&admin)
			logger.Info("Admin password was plain or invalid format, updated to hashed version.")
		}
	} else {
		// Create admin if doesn't exist
		hashed, _ := auth.HashPassword("admin")
		newAdmin := models.User{
			Username: "admin",
			Password: hashed,
			Role:     models.RoleAdmin,
		}
		if err := database.DB.Create(&newAdmin).Error; err != nil {
			logger.Error("Failed to create default admin user", "error", err)
		} else {
			logger.Info("Default admin user 'admin' created with password 'admin'.")
		}
	}

	var router *gin.Engine
	if logger.DebugMode {
		router = gin.Default()
	} else {
		router = gin.New()
		router.Use(gin.Recovery())
	}

	// Session management
	store := cookie.NewStore([]byte("secret-key-to-change")) // Replace with config secret in future
	router.Use(sessions.Sessions("flow_session", store))

	router.LoadHTMLGlob("templates/*")

	// Auth Routes
	router.GET("/login", handlers.ShowLogin)
	router.POST("/login", handlers.Login)
	router.GET("/logout", handlers.Logout)
	router.GET("/oidc/callback", handlers.OIDCCallback)

	// Protected Routes
	authorized := router.Group("/")
	authorized.Use(auth.AuthRequired(models.RoleViewer))
	{
		authorized.GET("/", handlers.ViewHandler)
		authorized.GET("/export", handlers.ExportHandler)
		authorized.GET("/vlan/lookup", handlers.VlanLookupHandler)
		authorized.GET("/vlan/export", handlers.ExportVlans)
		authorized.GET("/ci/lookup", handlers.CiLookupHandler)
		authorized.GET("/ci/suggest", handlers.CiSuggestHandler)
		authorized.GET("/ci/export", handlers.ExportCIs)

		// Requestor level
		requestor := authorized.Group("/")
		requestor.Use(auth.AuthRequired(models.RoleRequestor))
		{
			requestor.POST("/submit", handlers.SubmitHandler)
		}

		// Actor level
		actor := authorized.Group("/")
		actor.Use(auth.AuthRequired(models.RoleActor))
		{
			actor.POST("/vlan", handlers.CreateVlan)
			actor.PUT("/vlan/:id", handlers.UpdateVlan)
			actor.DELETE("/vlan/:id", handlers.DeleteVlan)
			actor.POST("/vlan/import", handlers.ImportVlans)

			actor.POST("/ci", handlers.CreateCI)
			actor.PUT("/ci/:id", handlers.UpdateCI)
			actor.DELETE("/ci/:id", handlers.DeleteCI)
			actor.POST("/ci/import", handlers.ImportCIs)

			actor.PUT("/flow/:id", handlers.UpdateFlow)
			actor.DELETE("/flow/:id", handlers.DeleteFlow)
		}

		// Admin only
		adminOnly := authorized.Group("/")
		adminOnly.Use(auth.AuthRequired(models.RoleAdmin))
		{
			adminOnly.POST("/users", handlers.CreateUser)
			adminOnly.PUT("/users/:id", handlers.UpdateUser)
			adminOnly.DELETE("/users/:id", handlers.DeleteUser)
		}
	}

	addr := fmt.Sprintf(":%d", config.Global.Server.Port)
	logger.Info("Starting server", "address", addr)
	if err := router.Run(addr); err != nil {
		logger.Fatal("Failed to start server", "error", err)
	}
}
