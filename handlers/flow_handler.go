package handlers

import (
	"flow-manager/logger"
	"flow-manager/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// UpdateFlow handles the update of an existing flow request (status, rule number, etc.)
func (h *Handler) UpdateFlow(c *gin.Context) {
	id := c.Param("id")
	var flow models.FlowRequest
	if err := h.DB.First(&flow, id).Error; err != nil {
		logger.Error("Flow not found for update", "id", id, "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Flow not found"})
		return
	}

	var input struct {
		RuleNumber string `json:"rule_number"`
		Status     string `json:"status"`
		Comment    string `json:"comment"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		logger.Warn("Failed to bind JSON for flow update", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Current User as Actor
	actorName := "System"
	if val, ok := c.Get("user"); ok {
		if u, ok := val.(models.User); ok {
			actorName = u.Username
		}
	}

	logger.Info("Updating flow", "id", id, "new_status", input.Status, "actor", actorName)

	// Update date on every action
	now := time.Now()
	flow.ImplementedAt = &now

	flow.RuleNumber = input.RuleNumber
	flow.Status = input.Status
	flow.Comment = input.Comment
	flow.LastActor = actorName

	if err := h.DB.Save(&flow).Error; err != nil {
		logger.Error("Failed to save updated flow", "id", id, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update flow: " + err.Error()})
		return
	}

	// Create history entry
	history := models.FlowHistory{
		FlowID:  flow.ID,
		Status:  input.Status,
		Actor:   actorName,
		Comment: input.Comment,
	}
	h.DB.Create(&history)

	c.JSON(http.StatusOK, flow)
}

// DeleteFlow handles the deletion of a flow request.
func (h *Handler) DeleteFlow(c *gin.Context) {
	id := c.Param("id")
	logger.Info("Deleting flow", "id", id)
	var flow models.FlowRequest
	if err := h.DB.First(&flow, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Flow not found"})
		return
	}

	if err := h.DB.Delete(&flow).Error; err != nil {
		logger.Error("Failed to delete flow", "id", id, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete flow"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Flow deleted successfully"})
}
