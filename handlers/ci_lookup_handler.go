package handlers

import (
	"flow-manager/database"
	"flow-manager/logger"
	"flow-manager/models"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CiLookupHandler handles the lookup of a CI by IP address or Hostname.
func (h *Handler) CiLookupHandler(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Query parameter is required"})
		return
	}

	logger.Debug("CI Lookup", "query", query)

	query = strings.ToLower(query)
	var ci models.CI
	// Search by IP or Hostname
	err := h.DB.Where("ip = ? OR hostname = ?", query, query).First(&ci).Error

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "CI not found"})
		return
	}

	var vlans []models.VlanSubnet
	if err := h.DB.Find(&vlans).Error; err == nil {
		ci.Vlan = database.MatchVLAN(ci.IP, vlans)
	}

	c.JSON(http.StatusOK, ci)
}

// CiSuggestHandler returns a list of CIs matching the prefix query.
func (h *Handler) CiSuggestHandler(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		c.JSON(http.StatusOK, []models.CI{})
		return
	}

	logger.Debug("CI Suggestion", "query", query)

	var cis []models.CI
	searchTerm := query + "%"
	h.DB.Limit(10).Where("hostname LIKE ? OR ip LIKE ?", searchTerm, searchTerm).Find(&cis)

	c.JSON(http.StatusOK, cis)
}
