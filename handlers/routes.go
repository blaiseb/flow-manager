package handlers

import (
	"flow-manager/auth"
	"flow-manager/models"
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SetupRoutes configures all the application routes.
func (h *Handler) SetupRoutes(router *gin.Engine, db *gorm.DB) {
	// Use embedded templates from handler
	templ := template.Must(template.New("").Funcs(router.FuncMap).ParseFS(h.TemplatesFS, "templates/*.html"))
	router.SetHTMLTemplate(templ)

	// Use embedded static files from handler
	router.Any("/static/*filepath", func(c *gin.Context) {
		fileServer := http.FileServer(http.FS(h.StaticFS))
		fileServer.ServeHTTP(c.Writer, c.Request)
	})

	// Auth Routes
	router.GET("/login", h.ShowLogin)
	router.POST("/login", h.Login)
	router.GET("/logout", h.Logout)
	router.GET("/oidc/callback", h.OIDCCallback)

	// Protected Routes
	authorized := router.Group("/")
	authorized.Use(auth.AuthRequired(db, models.RoleViewer))
	{
		authorized.GET("/", h.ViewHandler)
		authorized.GET("/export", h.ExportHandler)
		authorized.GET("/vlan/lookup", h.VlanLookupHandler)
		authorized.GET("/vlan/export", h.ExportVlans)
		authorized.GET("/ci/lookup", h.CiLookupHandler)
		authorized.GET("/ci/suggest", h.CiSuggestHandler)
		authorized.GET("/ci/export", h.ExportCIs)
		authorized.GET("/standard-flows", h.ListStandardFlows)

		// Requestor level
		requestor := authorized.Group("/")
		requestor.Use(auth.AuthRequired(db, models.RoleRequestor))
		{
			requestor.POST("/submit", h.SubmitHandler)
		}

		// Actor level
		actor := authorized.Group("/")
		actor.Use(auth.AuthRequired(db, models.RoleActor))
		{
			actor.POST("/vlan", h.CreateVlan)
			actor.PUT("/vlan/:id", h.UpdateVlan)
			actor.DELETE("/vlan/:id", h.DeleteVlan)
			actor.POST("/vlan/import", h.ImportVlans)

			actor.POST("/ci", h.CreateCI)
			actor.PUT("/ci/:id", h.UpdateCI)
			actor.DELETE("/ci/:id", h.DeleteCI)
			actor.POST("/ci/import", h.ImportCIs)

			actor.PUT("/flow/:id", h.UpdateFlow)
			actor.DELETE("/flow/:id", h.DeleteFlow)
		}

		// Admin only
		adminOnly := authorized.Group("/")
		adminOnly.Use(auth.AuthRequired(db, models.RoleAdmin))
		{
			adminOnly.POST("/users", h.CreateUser)
			adminOnly.PUT("/users/:id", h.UpdateUser)
			adminOnly.DELETE("/users/:id", h.DeleteUser)

			adminOnly.POST("/standard-flow", h.CreateStandardFlow)
			adminOnly.PUT("/standard-flow/:id", h.UpdateStandardFlow)
			adminOnly.DELETE("/standard-flow/:id", h.DeleteStandardFlow)
		}
	}
}
