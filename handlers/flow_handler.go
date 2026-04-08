package handlers

import (
	"net/http"
	"time"

	"flow-manager/database"
	"flow-manager/models"

	"github.com/gin-gonic/gin"
)

// UpdateFlow handles the update of an existing flow request (status, rule number, etc.)
func UpdateFlow(c *gin.Context) {
	id := c.Param("id")
	var flow models.FlowRequest
	if err := database.DB.First(&flow, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Flow not found"})
		return
	}

	var input struct {
		RuleNumber string `json:"rule_number"`
		Status     string `json:"status"`
		Comment    string `json:"comment"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Logic for ImplementedAt is also in BeforeUpdate hook, but let's be explicit
	// if status changes to "terminé"
	if input.Status == "terminé" && flow.Status != "terminé" {
		now := time.Now()
		flow.ImplementedAt = &now
	} else if input.Status != "terminé" {
		flow.ImplementedAt = nil
	}

	flow.RuleNumber = input.RuleNumber
	flow.Status = input.Status
	flow.Comment = input.Comment

	if err := database.DB.Save(&flow).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update flow: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, flow)
}

// DeleteFlow handles the deletion of a flow request.
func DeleteFlow(c *gin.Context) {
	id := c.Param("id")
	var flow models.FlowRequest
	if err := database.DB.First(&flow, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Flow not found"})
		return
	}

	if err := database.DB.Delete(&flow).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete flow"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Flow deleted successfully"})
}
