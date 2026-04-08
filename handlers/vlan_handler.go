package handlers

import (
	"encoding/csv"
	"io"
	"net/http"
	"strings"

	"flow-manager/database"
	"flow-manager/models"

	"github.com/gin-gonic/gin"
)

// ImportVlans handles bulk import of VLANs from a CSV file (comma-delimited).
func ImportVlans(c *gin.Context) {
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = ',' // Délimiteur virgule
	reader.FieldsPerRecord = -1 // Flexible number of fields

	var created, updated int
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue // Skip invalid rows
		}

		// skip header if present
		if len(record) > 0 && (strings.ToLower(record[0]) == "subnet") {
			continue
		}

		// Ensure we have at least 2 fields (subnet and name)
		if len(record) < 2 {
			continue
		}

		vlanData := models.VlanSubnet{
			Subnet: strings.TrimSpace(record[0]),
			VLAN:   strings.TrimSpace(record[1]),
		}
		if len(record) >= 3 {
			vlanData.Gateway = strings.TrimSpace(record[2])
		}
		if len(record) >= 4 {
			vlanData.DNSServers = strings.TrimSpace(record[3])
		}

		// Upsert logic based on subnet
		var existing models.VlanSubnet
		err = database.DB.Where("subnet = ?", vlanData.Subnet).First(&existing).Error
		if err == nil {
			// Update
			existing.VLAN = vlanData.VLAN
			existing.Gateway = vlanData.Gateway
			existing.DNSServers = vlanData.DNSServers
			database.DB.Save(&existing)
			updated++
		} else {
			// Create
			database.DB.Create(&vlanData)
			created++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Import completed",
		"created": created,
		"updated": updated,
	})
}

// ExportVlans exports all VLANs to a space-delimited CSV file.
func ExportVlans(c *gin.Context) {
	var vlans []models.VlanSubnet
	if err := database.DB.Find(&vlans).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch VLANs"})
		return
	}

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment;filename=vlans_export.csv")

	writer := csv.NewWriter(c.Writer)
	writer.Comma = ',' // Délimiteur virgule

	// En-tête
	writer.Write([]string{"subnet", "name", "gateway", "dns"})

	for _, v := range vlans {
		record := []string{
			v.Subnet,
			v.VLAN,
			v.Gateway,
			v.DNSServers,
		}
		writer.Write(record)
	}
	writer.Flush()
}

// CreateVlan handles the creation of a new VLAN.
func CreateVlan(c *gin.Context) {
	var vlan models.VlanSubnet
	if err := c.ShouldBindJSON(&vlan); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := database.DB.Create(&vlan).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create VLAN"})
		return
	}

	c.JSON(http.StatusOK, vlan)
}

// UpdateVlan handles the update of an existing VLAN.
func UpdateVlan(c *gin.Context) {
	id := c.Param("id")
	var vlan models.VlanSubnet
	if err := database.DB.First(&vlan, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "VLAN not found"})
		return
	}

	var updatedVlan models.VlanSubnet
	if err := c.ShouldBindJSON(&updatedVlan); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	vlan.Subnet = updatedVlan.Subnet
	vlan.VLAN = updatedVlan.VLAN
	vlan.Gateway = updatedVlan.Gateway
	vlan.DNSServers = updatedVlan.DNSServers

	if err := database.DB.Save(&vlan).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update VLAN"})
		return
	}

	c.JSON(http.StatusOK, vlan)
}

// DeleteVlan handles the deletion of a VLAN.
func DeleteVlan(c *gin.Context) {
	id := c.Param("id")
	var vlan models.VlanSubnet
	if err := database.DB.First(&vlan, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "VLAN not found"})
		return
	}

	if err := database.DB.Delete(&vlan).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete VLAN"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "VLAN deleted successfully"})
}
