package handlers

import (
	"net/http"

	"flow-manager/database"
	"flow-manager/models"

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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.IP == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "IP is required"})
		return
	}

	var ci models.CI
	err := database.DB.Unscoped().Where("ip = ?", input.IP).First(&ci).Error

	if err == nil {
		ci.DeletedAt = gorm.DeletedAt{}
		ci.FQDN = input.FQDN
		ci.Description = input.Description
		if err := database.DB.Unscoped().Save(&ci).Error; err != nil {
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

	ci.FQDN = input.FQDN
	ci.IP = input.IP
	ci.Description = input.Description

	if err := database.DB.Save(&ci).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update CI: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, ci)
}

// DeleteCI handles the deletion of a CI.
func DeleteCI(c *gin.Context) {
	id := c.Param("id")
	var ci models.CI
	if err := database.DB.First(&ci, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "CI not found"})
		return
	}

	if err := database.DB.Delete(&ci).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete CI"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "CI deleted successfully"})
}
