package handlers

import (
	"flow-manager/database"
	"flow-manager/logger"
	"flow-manager/models"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

// GenerateExcelFile produces an excelize.File from a slice of FlowRequests with merged headers.
func (h *Handler) GenerateExcelFile(flows []models.FlowRequest) (*excelize.File, error) {
	var vlans []models.VlanSubnet
	var cis []models.CI
	if err := h.DB.Find(&vlans).Error; err != nil {
		logger.Error("Failed to fetch VLANs for export", "error", err)
	}
	if err := h.DB.Find(&cis).Error; err != nil {
		logger.Error("Failed to fetch CIs for export", "error", err)
	}

	ciMap := make(map[string]models.CI)
	for _, ci := range cis {
		ciMap[ci.IP] = ci
	}

	parsedVlans := database.PreParseSubnets(vlans)

	f := excelize.NewFile()
	sheet := "Flux"
	index, err := f.NewSheet(sheet)
	if err != nil {
		return nil, fmt.Errorf("failed to create new sheet: %w", err)
	}
	f.SetActiveSheet(index)

	// Row 1: Merged Headers
	f.SetCellValue(sheet, "A1", "ID")
	f.MergeCell(sheet, "A1", "A2")
	f.SetCellValue(sheet, "B1", "Référence")
	f.MergeCell(sheet, "B1", "B2")
	
	f.SetCellValue(sheet, "C1", "SOURCE")
	f.MergeCell(sheet, "C1", "E1")
	
	f.SetCellValue(sheet, "F1", "CIBLE (TARGET)")
	f.MergeCell(sheet, "F1", "H1")
	
	f.SetCellValue(sheet, "I1", "Protocole")
	f.MergeCell(sheet, "I1", "I2")
	f.SetCellValue(sheet, "J1", "Port")
	f.MergeCell(sheet, "J1", "J2")
	f.SetCellValue(sheet, "K1", "Statut")
	f.MergeCell(sheet, "K1", "K2")
	f.SetCellValue(sheet, "L1", "Rule #")
	f.MergeCell(sheet, "L1", "L2")
	f.SetCellValue(sheet, "M1", "Date Action")
	f.MergeCell(sheet, "M1", "M2")
	f.SetCellValue(sheet, "N1", "Limite Temporelle")
	f.MergeCell(sheet, "N1", "N2")
	f.SetCellValue(sheet, "O1", "Commentaire")
	f.MergeCell(sheet, "O1", "O2")
	f.SetCellValue(sheet, "P1", "Date Création")
	f.MergeCell(sheet, "P1", "P2")

	// Row 2: Sub-headers
	f.SetCellValue(sheet, "C2", "VLAN")
	f.SetCellValue(sheet, "D2", "IP / Subnet")
	f.SetCellValue(sheet, "E2", "Hostname")
	f.SetCellValue(sheet, "F2", "VLAN")
	f.SetCellValue(sheet, "G2", "IP / Subnet")
	f.SetCellValue(sheet, "H2", "Hostname")

	// Styles for headers
	style, err := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#E0E0E0"}, Pattern: 1},
		Font: &excelize.Font{Bold: true},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	if err != nil {
		logger.Error("Failed to create style for export", "error", err)
	} else {
		f.SetCellStyle(sheet, "A1", "P2", style)
	}

	for i, flow := range flows {
		row := i + 3 // Data starts at row 3
		
		timeLimit := "Sans limite"
		if flow.TimeLimit != nil {
			timeLimit = flow.TimeLimit.Format("2006-01-02")
		}

		actionDate := "-"
		if flow.ImplementedAt != nil {
			actionDate = flow.ImplementedAt.Format("2006-01-02")
		}

		srcHostname := "-"
		if flow.SourceHostname != "" {
			srcHostname = flow.SourceHostname
		} else if ci, ok := ciMap[flow.SourceIP]; ok {
			srcHostname = ci.Hostname
		}
		srcVlanName := "Inconnu"
		if v := database.MatchVLANOptimized(flow.SourceIP, parsedVlans); v != nil {
			srcVlanName = v.VLAN
		}

		tgtHostname := "-"
		if flow.TargetHostname != "" {
			tgtHostname = flow.TargetHostname
		} else if ci, ok := ciMap[flow.TargetIP]; ok {
			tgtHostname = ci.Hostname
		}
		tgtVlanName := "Inconnu"
		if v := database.MatchVLANOptimized(flow.TargetIP, parsedVlans); v != nil {
			tgtVlanName = v.VLAN
		}

		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), flow.ID)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), flow.Reference)
		
		// Source
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), srcVlanName)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), flow.SourceIP)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), srcHostname)
		
		// Target
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), tgtVlanName)
		f.SetCellValue(sheet, fmt.Sprintf("G%d", row), flow.TargetIP)
		f.SetCellValue(sheet, fmt.Sprintf("H%d", row), tgtHostname)
		
		f.SetCellValue(sheet, fmt.Sprintf("I%d", row), flow.Protocol)
		f.SetCellValue(sheet, fmt.Sprintf("J%d", row), flow.Port)
		f.SetCellValue(sheet, fmt.Sprintf("K%d", row), flow.Status)
		f.SetCellValue(sheet, fmt.Sprintf("L%d", row), flow.RuleNumber)
		f.SetCellValue(sheet, fmt.Sprintf("M%d", row), actionDate)
		f.SetCellValue(sheet, fmt.Sprintf("N%d", row), timeLimit)
		f.SetCellValue(sheet, fmt.Sprintf("O%d", row), flow.Comment)
		f.SetCellValue(sheet, fmt.Sprintf("P%d", row), flow.CreatedAt.Format("2006-01-02 15:04:05"))
	}
	f.DeleteSheet("Sheet1")
	return f, nil
}

// ExportHandler generates an Excel file of all flow requests.
func (h *Handler) ExportHandler(c *gin.Context) {
	logger.Info("Generating complete flows export")
	var flows []models.FlowRequest
	h.DB.Order("created_at desc").Find(&flows)

	f, err := h.GenerateExcelFile(flows)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate Excel file"})
		return
	}

	fileName := fmt.Sprintf("export_complet_%s.xlsx", time.Now().Format("20060102_150405"))
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", "attachment; filename="+fileName)
	if err := f.Write(c.Writer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to write Excel file"})
	}
}
