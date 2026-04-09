package handlers

import (
	"embed"
	"gorm.io/gorm"
)

type Handler struct {
	DB          *gorm.DB
	StaticFS    embed.FS
	TemplatesFS embed.FS
}

func NewHandler(db *gorm.DB, static embed.FS, templates embed.FS) *Handler {
	return &Handler{
		DB:          db,
		StaticFS:    static,
		TemplatesFS: templates,
	}
}
