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
	csrf "github.com/utrack/gin-csrf"
)

func main() {
	// Parsing flags
	configPath := flag.String("config", "config.yaml", "Path to config file")
	debugFlag := flag.Bool("debug", false, "Force enable debug mode (overrides config)")
	portFlag := flag.Int("port", 0, "Server port (overrides config)")
	flag.Parse()

	// Load Logger first (using default level until config is loaded)
	logFile, _ := os.OpenFile("flow-manager.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	logger.InitLogger(logFile, "info")

	// Load Configuration
	if err := config.LoadConfig(*configPath); err != nil {
		logger.Warn("Could not load config file, using defaults", "error", err)
	}

	if *debugFlag {
		config.Global.Log.Debug = true
		config.Global.Log.Level = "debug"
	}
	if *portFlag != 0 {
		config.Global.Server.Port = *portFlag
	}

	logger.DebugMode = config.Global.Log.Debug

	// Re-init Logger with proper level from config
	logger.InitLogger(logFile, config.Global.Log.Level)

	if logger.DebugMode {
		logger.Debug("Debug mode is enabled")
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	gin.DefaultWriter = io.MultiWriter(logFile, os.Stdout)

	db := database.InitDatabase()
	handlers.InitOIDC()
	h := handlers.NewHandler(db)

	// Ensure admin user has hashed password
	var admin models.User
	if err := db.Where("username = ?", "admin").First(&admin).Error; err == nil {
		// If password is still plain "admin", hash it
		if admin.Password == "admin" || !strings.HasPrefix(admin.Password, "$2a$") {
			initialPassword := os.Getenv("INITIAL_ADMIN_PASSWORD")
			if initialPassword == "" {
				initialPassword = "admin"
				logger.Warn("Admin password is still 'admin' and no INITIAL_ADMIN_PASSWORD found. Please change it immediately.")
			}
			hashed, err := auth.HashPassword(initialPassword)
			if err != nil {
				logger.Fatal("Failed to hash admin password", "error", err)
			}
			admin.Password = hashed
			db.Save(&admin)
			logger.Info("Admin password was plain or invalid format, updated to hashed version.")
		}
	} else {
		// Create admin if doesn't exist
		initialPassword := os.Getenv("INITIAL_ADMIN_PASSWORD")
		if initialPassword == "" {
			initialPassword = "admin"
			logger.Warn("Creating default admin user 'admin' with password 'admin'. Please set INITIAL_ADMIN_PASSWORD next time.")
		}
		hashed, err := auth.HashPassword(initialPassword)
		if err != nil {
			logger.Fatal("Failed to hash default admin password", "error", err)
		}
		newAdmin := models.User{
			Username: "admin",
			Password: hashed,
			Role:     models.RoleAdmin,
		}
		if err := db.Create(&newAdmin).Error; err != nil {
			logger.Error("Failed to create default admin user", "error", err)
		} else {
			logger.Info("Default admin user 'admin' created.")
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
	sessionSecret := config.Global.Server.Secret
	if sessionSecret == "" {
		if !logger.DebugMode {
			logger.Fatal("FLOW_SESSION_SECRET or server.secret config is missing!")
		}
		sessionSecret = "dev-secret-key"
		logger.Warn("Using insecure session secret in debug mode")
	}
	store := cookie.NewStore([]byte(sessionSecret))
	router.Use(sessions.Sessions("flow_session", store))

	// CSRF Protection
	router.Use(csrf.Middleware(csrf.Options{
		Secret: sessionSecret,
		ErrorFunc: func(c *gin.Context) {
			logger.Warn("CSRF error detected", "ip", c.ClientIP(), "path", c.Request.URL.Path)
			c.String(400, "CSRF error")
			c.Abort()
		},
	}))

	// Inject CSRF token into template context
	router.Use(func(c *gin.Context) {
		c.Set("_csrf", csrf.GetToken(c))
		c.Next()
	})

	h.SetupRoutes(router, db)

	addr := fmt.Sprintf(":%d", config.Global.Server.Port)
	logger.Info("Starting server", "address", addr)
	if err := router.Run(addr); err != nil {
		logger.Fatal("Failed to start server", "error", err)
	}
}
