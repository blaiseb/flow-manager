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

// GenerateExcelFile produces an excelize.File from a slice of FlowRequests.
func GenerateExcelFile(flows []models.FlowRequest) (*excelize.File, error) {
	var vlans []models.VlanSubnet
	var cis []models.CI
	database.DB.Find(&vlans)
	database.DB.Find(&cis)

	ciMap := make(map[string]models.CI)
	for _, ci := range cis {
		ciMap[ci.IP] = ci
	}

	f := excelize.NewFile()
	sheet := "Flux"
	index, _ := f.NewSheet(sheet)
	f.SetActiveSheet(index)

	headers := []string{
		"ID", "Référence", "Source FQDN", "Source IP", "Source VLAN", "Cible FQDN", "Cible IP", "Cible VLAN",
		"Protocole", "Port", "Statut", "Rule #", "Date Action", "Limite Temporelle", "Commentaire", "Date Création",
	}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, header)
	}

	for i, flow := range flows {
		row := i + 2
		
		timeLimit := "Sans limite"
		if flow.TimeLimit != nil {
			timeLimit = flow.TimeLimit.Format("2006-01-02")
		}

		actionDate := "-"
		if flow.ImplementedAt != nil {
			actionDate = flow.ImplementedAt.Format("2006-01-02")
		}

		srcFQDN := "-"
		if ci, ok := ciMap[flow.SourceIP]; ok {
			srcFQDN = ci.FQDN
		}
		srcVlanName := "Inconnu"
		if v := database.MatchVLAN(flow.SourceIP, vlans); v != nil {
			srcVlanName = v.VLAN
		}

		tgtFQDN := "-"
		if ci, ok := ciMap[flow.TargetIP]; ok {
			tgtFQDN = ci.FQDN
		}
		tgtVlanName := "Inconnu"
		if v := database.MatchVLAN(flow.TargetIP, vlans); v != nil {
			tgtVlanName = v.VLAN
		}

		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), flow.ID)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), flow.Reference)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), srcFQDN)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), flow.SourceIP)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), srcVlanName)
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), tgtFQDN)
		f.SetCellValue(sheet, fmt.Sprintf("G%d", row), flow.TargetIP)
		f.SetCellValue(sheet, fmt.Sprintf("H%d", row), tgtVlanName)
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
func ExportHandler(c *gin.Context) {
	logger.Info("Generating complete flows export")
	var flows []models.FlowRequest
	database.DB.Order("created_at desc").Find(&flows)

	f, err := GenerateExcelFile(flows)
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
