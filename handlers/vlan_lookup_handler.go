package handlers

import (
	"flow-manager/database"
	"flow-manager/logger"
	"net/http"

	"github.com/gin-gonic/gin"
)

// VlanLookupHandler handles the lookup of a VLAN by IP address.
func VlanLookupHandler(c *gin.Context) {
	ipQuery := c.Query("ip")
	if ipQuery == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "IP address is required"})
		return
	}

	logger.Debug("VLAN Lookup", "ip", ipQuery)

	vlan, err := database.FindVLAN(ipQuery)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"vlan": vlan})
}
