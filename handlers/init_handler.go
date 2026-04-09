package handlers

import (
	"encoding/csv"
	"flow-manager/logger"
	"flow-manager/models"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gorm.io/gorm"
)

// AutoImport checks if the database is empty and imports CSV files from init-data directory.
func (h *Handler) AutoImport() {
	initDir := "init-data"
	if _, err := os.Stat(initDir); os.IsNotExist(err) {
		return
	}

	// 1. Import VLANs if table is empty
	var vlanCount int64
	h.DB.Model(&models.VlanSubnet{}).Count(&vlanCount)
	if vlanCount == 0 {
		vlanFile := filepath.Join(initDir, "vlans.csv")
		if _, err := os.Stat(vlanFile); err == nil {
			h.autoImportVlans(vlanFile)
		}
	}

	// 2. Import CIs if table is empty
	var ciCount int64
	h.DB.Model(&models.CI{}).Count(&ciCount)
	if ciCount == 0 {
		ciFile := filepath.Join(initDir, "cis.csv")
		if _, err := os.Stat(ciFile); err == nil {
			h.autoImportCIs(ciFile)
		}
	}
}

func (h *Handler) autoImportVlans(path string) {
	file, err := os.Open(path)
	if err != nil {
		logger.Error("Failed to open VLAN init file", "path", path, "error", err)
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = ','
	reader.FieldsPerRecord = -1

	var created, failed int
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
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

		if err := h.DB.Create(&vlanData).Error; err != nil {
			failed++
		} else {
			created++
		}
	}
	logger.Info("Auto-import VLANs completed", "path", path, "created", created, "failed", failed)
}

func (h *Handler) autoImportCIs(path string) {
	file, err := os.Open(path)
	if err != nil {
		logger.Error("Failed to open CI init file", "path", path, "error", err)
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = ','
	reader.FieldsPerRecord = -1

	// Read header
	header, err := reader.Read()
	if err != nil {
		return
	}

	colMap := make(map[string]int)
	for i, col := range header {
		col = strings.ToLower(strings.TrimSpace(col))
		if col == "fqdn" {
			col = "hostname"
		}
		colMap[col] = i
	}

	ipIdx, hasIP := colMap["ip"]
	hostnameIdx, hasHostname := colMap["hostname"]
	if !hasIP || !hasHostname {
		ipIdx = 0
		hostnameIdx = 1
	}

	descIdx, hasDesc := colMap["description"]
	if !hasDesc {
		descIdx = 2
	}

	var created, failed int
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			failed++
			continue
		}

		if len(record) <= hostnameIdx || len(record) <= ipIdx {
			failed++
			continue
		}

		ip := strings.TrimSpace(record[ipIdx])
		hostname := strings.TrimSpace(record[hostnameIdx])
		if ip == "" {
			failed++
			continue
		}

		description := ""
		if hasDesc && len(record) > descIdx {
			description = strings.TrimSpace(record[descIdx])
		}

		newCi := models.CI{
			Hostname:    hostname,
			IP:          ip,
			Description: description,
		}
		if err := h.DB.Create(&newCi).Error; err != nil {
			// If already exists (e.g. duplicate in CSV), skip or update
			var existing models.CI
			if err := h.DB.Unscoped().Where("ip = ?", ip).First(&existing).Error; err == nil {
				existing.DeletedAt = gorm.DeletedAt{}
				existing.Hostname = hostname
				existing.Description = description
				h.DB.Unscoped().Save(&existing)
			}
			failed++
		} else {
			created++
		}
	}
	logger.Info("Auto-import CIs completed", "path", path, "created", created, "failed", failed)
}
