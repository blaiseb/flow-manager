package handlers

import (
	"encoding/csv"
	"flow-manager/database"
	"flow-manager/logger"
	"flow-manager/models"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ImportCIs handles bulk import of CIs from a CSV file (comma-delimited).
// Identifies columns by reading the header line.
func (h *Handler) ImportCIs(c *gin.Context) {
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		logger.Warn("No file uploaded for CI import")
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = ','
	reader.FieldsPerRecord = -1

	// Read header
	header, err := reader.Read()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read CSV header"})
		return
	}

	// Map columns
	colMap := make(map[string]int)
	for i, col := range header {
		col = strings.ToLower(strings.TrimSpace(col))
		// Handle both 'hostname' and 'fqdn' for backward compatibility during import
		if col == "fqdn" {
			col = "hostname"
		}
		colMap[col] = i
	}

	// Required columns
	ipIdx, hasIP := colMap["ip"]
	hostnameIdx, hasHostname := colMap["hostname"]

	if !hasIP || !hasHostname {
		// Fallback to default order if header is not recognized
		// IP, Hostname, Description, VLAN
		ipIdx = 0
		hostnameIdx = 1
		logger.Info("CSV header not fully recognized, falling back to default order: IP, Hostname, ...")
	}

	descIdx, hasDesc := colMap["description"]
	if !hasDesc {
		descIdx = 2
	}

	var created, updated int
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			logger.Warn("Error reading CSV record during CI import", "error", err)
			continue
		}

		if len(record) <= hostnameIdx || len(record) <= ipIdx {
			continue
		}

		ip := strings.TrimSpace(record[ipIdx])
		hostname := strings.TrimSpace(record[hostnameIdx])
		description := ""
		if hasDesc && len(record) > descIdx {
			description = strings.TrimSpace(record[descIdx])
		} else if !hasDesc && len(record) > 2 {
			description = strings.TrimSpace(record[2])
		}

		if ip == "" {
			continue
		}

		var existing models.CI
		err = h.DB.Unscoped().Where("ip = ?", ip).First(&existing).Error
		if err == nil {
			existing.DeletedAt = gorm.DeletedAt{}
			existing.Hostname = hostname
			existing.Description = description
			if err := h.DB.Unscoped().Save(&existing).Error; err != nil {
				logger.Error("Failed to update CI during import", "ip", ip, "error", err)
			} else {
				updated++
			}
		} else {
			newCi := models.CI{
				Hostname:    hostname,
				IP:          ip,
				Description: description,
			}
			if err := h.DB.Create(&newCi).Error; err != nil {
				logger.Error("Failed to create CI during import", "ip", ip, "error", err)
			} else {
				created++
			}
		}
	}

	logger.Info("CI import completed", "created", created, "updated", updated)
	c.JSON(http.StatusOK, gin.H{"message": "Import completed", "created": created, "updated": updated})
}

// ExportCIs exports all CIs to a comma-delimited CSV file.
// Order: IP, Hostname, Description, VLAN
func (h *Handler) ExportCIs(c *gin.Context) {
	var cis []models.CI
	var vlans []models.VlanSubnet
	if err := h.DB.Find(&vlans).Error; err != nil {
		logger.Error("Failed to fetch VLANs for CI export", "error", err)
	}
	if err := h.DB.Find(&cis).Error; err != nil {
		logger.Error("Failed to fetch CIs for export", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch CIs"})
		return
	}

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment;filename=cis_export.csv")

	writer := csv.NewWriter(c.Writer)
	writer.Comma = ','

	writer.Write([]string{"ip", "hostname", "description", "vlan"})

	for _, ci := range cis {
		vlanName := "Inconnu"
		if v := database.MatchVLAN(ci.IP, vlans); v != nil {
			vlanName = v.VLAN
		}
		writer.Write([]string{
			ci.IP,
			ci.Hostname,
			ci.Description,
			vlanName,
		})
	}
	writer.Flush()
}

// CreateCI handles the creation of a new CI.
func (h *Handler) CreateCI(c *gin.Context) {
	var input struct {
		Hostname    string `json:"hostname"`
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

	logger.Info("Creating/Restoring CI", "ip", input.IP, "hostname", input.Hostname)

	var ci models.CI
	err := h.DB.Unscoped().Where("ip = ?", input.IP).First(&ci).Error

	if err == nil {
		ci.DeletedAt = gorm.DeletedAt{}
		ci.Hostname = input.Hostname
		ci.Description = input.Description
		if err := h.DB.Unscoped().Save(&ci).Error; err != nil {
			logger.Error("Failed to restore CI", "ip", input.IP, "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to restore CI: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, ci)
		return
	}

	ci = models.CI{
		Hostname:    input.Hostname,
		IP:          input.IP,
		Description: input.Description,
	}

	if err := h.DB.Create(&ci).Error; err != nil {
		logger.Error("Failed to create CI", "ip", input.IP, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create CI: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, ci)
}

// UpdateCI handles the update of an existing CI.
func (h *Handler) UpdateCI(c *gin.Context) {
	id := c.Param("id")
	var ci models.CI
	if err := h.DB.First(&ci, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "CI not found"})
		return
	}

	var input struct {
		Hostname    string `json:"hostname"`
		IP          string `json:"ip"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	logger.Info("Updating CI", "id", id, "ip", input.IP)

	ci.Hostname = input.Hostname
	ci.IP = input.IP
	ci.Description = input.Description

	if err := h.DB.Save(&ci).Error; err != nil {
		logger.Error("Failed to update CI", "id", id, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update CI: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, ci)
}

// DeleteCI handles the deletion of a CI.
func (h *Handler) DeleteCI(c *gin.Context) {
	id := c.Param("id")
	logger.Info("Deleting CI", "id", id)
	var ci models.CI
	if err := h.DB.First(&ci, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "CI not found"})
		return
	}

	if err := h.DB.Delete(&ci).Error; err != nil {
		logger.Error("Failed to delete CI", "id", id, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete CI"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "CI deleted successfully"})
}
