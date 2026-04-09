package handlers

import (
	"encoding/csv"
	"flow-manager/logger"
	"flow-manager/models"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// ImportVlans handles bulk import of VLANs from a CSV file (comma-delimited).
func (h *Handler) ImportVlans(c *gin.Context) {
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		logger.Warn("No file uploaded for VLAN import")
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = ','
	reader.FieldsPerRecord = -1

	var created, updated, failed int
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			logger.Warn("Error reading CSV record during VLAN import", "error", err)
			failed++
			continue
		}

		if len(record) > 0 && (strings.ToLower(strings.TrimSpace(record[0])) == "subnet") {
			continue
		}

		if len(record) < 2 {
			failed++
			continue
		}

		subnet := strings.TrimSpace(record[0])
		if subnet == "" {
			failed++
			continue
		}

		vlanData := models.VlanSubnet{
			Subnet: subnet,
			VLAN:   strings.TrimSpace(record[1]),
		}
		if len(record) >= 3 {
			vlanData.Gateway = strings.TrimSpace(record[2])
		}
		if len(record) >= 4 {
			vlanData.DNSServers = strings.TrimSpace(record[3])
		}

		var existing models.VlanSubnet
		err = h.DB.Where("subnet = ?", vlanData.Subnet).First(&existing).Error
		if err == nil {
			existing.VLAN = vlanData.VLAN
			existing.Gateway = vlanData.Gateway
			existing.DNSServers = vlanData.DNSServers
			if err := h.DB.Save(&existing).Error; err != nil {
				logger.Error("Failed to update VLAN during import", "subnet", subnet, "error", err)
				failed++
			} else {
				updated++
			}
		} else {
			if err := h.DB.Create(&vlanData).Error; err != nil {
				logger.Error("Failed to create VLAN during import", "subnet", subnet, "error", err)
				failed++
			} else {
				created++
			}
		}
	}

	logger.Info("VLAN import completed", "created", created, "updated", updated, "failed", failed)

	c.JSON(http.StatusOK, gin.H{
		"message": "Import terminé",
		"created": created,
		"updated": updated,
		"failed":  failed,
	})
}

// ExportVlans exports all VLANs to a comma-delimited CSV file.
func (h *Handler) ExportVlans(c *gin.Context) {
	var vlans []models.VlanSubnet
	if err := h.DB.Find(&vlans).Error; err != nil {
		logger.Error("Failed to fetch VLANs for export", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch VLANs"})
		return
	}

	logger.Info("Exporting VLANs", "count", len(vlans))

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment;filename=vlans_export.csv")

	writer := csv.NewWriter(c.Writer)
	writer.Comma = ','

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
func (h *Handler) CreateVlan(c *gin.Context) {
	var vlan models.VlanSubnet
	if err := c.ShouldBindJSON(&vlan); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	logger.Info("Creating VLAN", "vlan", vlan.VLAN, "subnet", vlan.Subnet)

	if err := h.DB.Create(&vlan).Error; err != nil {
		logger.Error("Failed to create VLAN", "vlan", vlan.VLAN, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create VLAN"})
		return
	}

	c.JSON(http.StatusOK, vlan)
}

// UpdateVlan handles the update of an existing VLAN.
func (h *Handler) UpdateVlan(c *gin.Context) {
	id := c.Param("id")
	var vlan models.VlanSubnet
	if err := h.DB.First(&vlan, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "VLAN not found"})
		return
	}

	var updatedVlan models.VlanSubnet
	if err := c.ShouldBindJSON(&updatedVlan); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	logger.Info("Updating VLAN", "id", id, "vlan", vlan.VLAN)

	vlan.Subnet = updatedVlan.Subnet
	vlan.VLAN = updatedVlan.VLAN
	vlan.Gateway = updatedVlan.Gateway
	vlan.DNSServers = updatedVlan.DNSServers

	if err := h.DB.Save(&vlan).Error; err != nil {
		logger.Error("Failed to update VLAN", "id", id, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update VLAN"})
		return
	}

	c.JSON(http.StatusOK, vlan)
}

// DeleteVlan handles the deletion of a VLAN.
func (h *Handler) DeleteVlan(c *gin.Context) {
	id := c.Param("id")
	logger.Info("Deleting VLAN", "id", id)
	var vlan models.VlanSubnet
	if err := h.DB.First(&vlan, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "VLAN not found"})
		return
	}

	if err := h.DB.Delete(&vlan).Error; err != nil {
		logger.Error("Failed to delete VLAN", "id", id, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete VLAN"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "VLAN deleted successfully"})
}
