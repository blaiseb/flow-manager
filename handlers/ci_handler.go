package handlers

import (
	"flow-manager/database"
	"flow-manager/logger"
	"flow-manager/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CreateCI handles the creation of a new CI.
func CreateCI(c *gin.Context) {
	var input struct {
		FQDN        string `json:"fqdn"`
		IP          string `json:"ip"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		logger.Warn("Failed to bind JSON for CI creation", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.IP == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "IP is required"})
		return
	}

	logger.Info("Creating/Restoring CI", "ip", input.IP, "fqdn", input.FQDN)

	var ci models.CI
	err := database.DB.Unscoped().Where("ip = ?", input.IP).First(&ci).Error

	if err == nil {
		ci.DeletedAt = gorm.DeletedAt{}
		ci.FQDN = input.FQDN
		ci.Description = input.Description
		if err := database.DB.Unscoped().Save(&ci).Error; err != nil {
			logger.Error("Failed to restore CI", "ip", input.IP, "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to restore CI: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, ci)
		return
	}

	ci = models.CI{
		FQDN:        input.FQDN,
		IP:          input.IP,
		Description: input.Description,
	}

	if err := database.DB.Create(&ci).Error; err != nil {
		logger.Error("Failed to create CI", "ip", input.IP, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create CI: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, ci)
}

// UpdateCI handles the update of an existing CI.
func UpdateCI(c *gin.Context) {
	id := c.Param("id")
	var ci models.CI
	if err := database.DB.First(&ci, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "CI not found"})
		return
	}

	var input struct {
		FQDN        string `json:"fqdn"`
		IP          string `json:"ip"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	logger.Info("Updating CI", "id", id, "ip", input.IP)

	ci.FQDN = input.FQDN
	ci.IP = input.IP
	ci.Description = input.Description

	if err := database.DB.Save(&ci).Error; err != nil {
		logger.Error("Failed to update CI", "id", id, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update CI: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, ci)
}

// DeleteCI handles the deletion of a CI.
func DeleteCI(c *gin.Context) {
	id := c.Param("id")
	logger.Info("Deleting CI", "id", id)
	var ci models.CI
	if err := database.DB.First(&ci, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "CI not found"})
		return
	}

	if err := database.DB.Delete(&ci).Error; err != nil {
		logger.Error("Failed to delete CI", "id", id, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete CI"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "CI deleted successfully"})
}
