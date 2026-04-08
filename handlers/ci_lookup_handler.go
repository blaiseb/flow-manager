package handlers

import (
	"net/http"
	"strings"

	"flow-manager/database"
	"flow-manager/models"

	"github.com/gin-gonic/gin"
)

// CiLookupHandler handles the lookup of a CI by IP address or FQDN.
func CiLookupHandler(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Query parameter is required"})
		return
	}

	query = strings.ToLower(query)
	var ci models.CI
	// Search by IP or FQDN
	err := database.DB.Where("ip = ? OR fqdn = ?", query, query).First(&ci).Error

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "CI not found"})
		return
	}

	// Dynamic VLAN calculation for the lookup response
	var vlans []models.VlanSubnet
	if err := database.DB.Find(&vlans).Error; err == nil {
		ci.Vlan = database.MatchVLAN(ci.IP, vlans)
	}

	c.JSON(http.StatusOK, ci)
}

// CiSuggestHandler returns a list of CIs matching the prefix query.
func CiSuggestHandler(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		c.JSON(http.StatusOK, []models.CI{})
		return
	}

	var cis []models.CI
	searchTerm := query + "%"
	database.DB.Limit(10).Where("fqdn LIKE ? OR ip LIKE ?", searchTerm, searchTerm).Find(&cis)

	c.JSON(http.StatusOK, cis)
}
