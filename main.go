package main

import (
	"embed"
	"encoding/json"
	"flag"
	"flow-manager/config"
	"flow-manager/database"
	"flow-manager/handlers"
	"flow-manager/logger"
	"fmt"
	"html/template"
	"io"
	"os"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	csrf "github.com/utrack/gin-csrf"
)

//go:embed static/*
var staticFS embed.FS

//go:embed templates/*
var templatesFS embed.FS

const Version = "0.0.1"

func main() {
	// Parsing flags
	configPath := flag.String("config", "config.yaml", "Path to config file")
	debugFlag := flag.Bool("debug", false, "Force enable debug mode (overrides config)")
	portFlag := flag.Int("port", 0, "Server port (overrides config)")
	versionFlag := flag.Bool("version", false, "Show version and exit")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("Flow Manager version %s\n", Version)
		return
	}

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
	database.SeedDefaultData(db)
	
	handlers.InitOIDC()
	h := handlers.NewHandler(db, staticFS, templatesFS)
	h.AutoImport()

	var router *gin.Engine
	if logger.DebugMode {
		router = gin.Default()
	} else {
		router = gin.New()
		router.Use(gin.Recovery())
	}

	// Register custom template functions
	router.SetFuncMap(template.FuncMap{
		"JSONMarshal": func(v interface{}) template.JS {
			a, _ := json.Marshal(v)
			return template.JS(a)
		},
	})

	// Session management
	sessionSecret := config.Global.Server.Secret
	if sessionSecret == "" {
		sessionSecret = "dev-secret-key-change-me-in-production"
		if os.Getenv("GIN_MODE") == "release" {
			logger.Fatal("FLOW_SESSION_SECRET or server.secret config is missing! Required in release mode.")
		} else {
			logger.Warn("FLOW_SESSION_SECRET is missing. Using an insecure default secret. DO NOT USE IN PRODUCTION!")
		}
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
