package handlers

import (
	"flow-manager/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ListStandardFlows returns all standard flows.
func (h *Handler) ListStandardFlows(c *gin.Context) {
	var flows []models.StandardFlow
	if err := h.DB.Find(&flows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch standard flows"})
		return
	}
	c.JSON(http.StatusOK, flows)
}

// CreateStandardFlow creates a new standard flow preset.
func (h *Handler) CreateStandardFlow(c *gin.Context) {
	var input models.StandardFlow
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.DB.Create(&input).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create standard flow"})
		return
	}
	c.JSON(http.StatusOK, input)
}

// UpdateStandardFlow updates an existing standard flow preset.
func (h *Handler) UpdateStandardFlow(c *gin.Context) {
	id := c.Param("id")
	var flow models.StandardFlow
	if err := h.DB.First(&flow, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Standard flow not found"})
		return
	}

	var input models.StandardFlow
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	flow.Name = input.Name
	flow.Protocol = input.Protocol
	flow.Ports = input.Ports

	if err := h.DB.Save(&flow).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update standard flow"})
		return
	}
	c.JSON(http.StatusOK, flow)
}

// DeleteStandardFlow deletes a standard flow preset.
func (h *Handler) DeleteStandardFlow(c *gin.Context) {
	id := c.Param("id")
	if err := h.DB.Delete(&models.StandardFlow{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete standard flow"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Standard flow deleted"})
}
