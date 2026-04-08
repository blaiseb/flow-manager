package handlers

import (
	"flow-manager/database"
	"flow-manager/logger"
	"flow-manager/models"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type FlowSubmission struct {
	SourceFQDN string `form:"source_fqdn"`
	SourceIP   string `form:"source_ip"`
	TargetFQDN string `form:"target_fqdn"`
	TargetIP   string `form:"target_ip"`
	Protocol   string `form:"protocol"`
	Ports      string `form:"ports"`
	TimeLimit  string `form:"time_limit"`
	Comment    string `form:"comment"`
}

func SubmitHandler(c *gin.Context) {
	if err := c.Request.ParseForm(); err != nil {
		logger.Error("Failed to parse submission form", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse form"})
		return
	}

	action := c.PostForm("action") // "generate" or "validate"
	reference := "REF-" + time.Now().Format("20060102-150405")

	// Group fields by row index
	rows := make(map[string]*FlowSubmission)
	for key, values := range c.Request.PostForm {
		if !strings.HasPrefix(key, "flows[") {
			continue
		}
		idx := key[6:strings.Index(key, "]")]
		if _, ok := rows[idx]; !ok {
			rows[idx] = &FlowSubmission{}
		}
		val := values[0]
		field := key[strings.Index(key, "].")+2:]
		switch field {
		case "source_fqdn": rows[idx].SourceFQDN = val
		case "source_ip":   rows[idx].SourceIP = val
		case "target_fqdn": rows[idx].TargetFQDN = val
		case "target_ip":   rows[idx].TargetIP = val
		case "protocol":    rows[idx].Protocol = val
		case "ports":       rows[idx].Ports = val
		case "time_limit":  rows[idx].TimeLimit = val
		case "comment":     rows[idx].Comment = val
		}
	}

	var flowsToExport []models.FlowRequest
	
	for _, sub := range rows {
		ports := parsePorts(sub.Ports)
		if len(ports) == 0 {
			ports = []int{0}
		}

		var timeLimit *time.Time
		if sub.TimeLimit != "" {
			t, err := time.Parse("2006-01-02", sub.TimeLimit)
			if err == nil {
				timeLimit = &t
			}
		}

		for _, port := range ports {
			flow := models.FlowRequest{
				SourceIP:   sub.SourceIP,
				TargetIP:   sub.TargetIP,
				Protocol:   sub.Protocol,
				Port:       port,
				TimeLimit:  timeLimit,
				Comment:    sub.Comment,
				Reference:  reference,
				Status:     "demandé",
			}
			
			if action == "validate" {
				if err := database.DB.Create(&flow).Error; err != nil {
					logger.Error("Failed to create flow request", "error", err)
				}
			}
			
			// We need FQDNs for the Excel generation (even if not saved)
			flow.SourceFQDN = sub.SourceFQDN
			flow.TargetFQDN = sub.TargetFQDN
			
			flowsToExport = append(flowsToExport, flow)
		}
		
		if action == "validate" {
			if sub.SourceFQDN != "" && sub.SourceIP != "" && !strings.Contains(sub.SourceIP, "/") {
				ensureCI(sub.SourceFQDN, sub.SourceIP)
			}
			if sub.TargetFQDN != "" && sub.TargetIP != "" && !strings.Contains(sub.TargetIP, "/") {
				ensureCI(sub.TargetFQDN, sub.TargetIP)
			}
		}
	}

	if action == "generate" {
		logger.Info("Generating Excel request (preview)", "count", len(flowsToExport))
		f, err := GenerateExcelFile(flowsToExport)
		if err == nil {
			fileName := fmt.Sprintf("demande_draft_%s.xlsx", reference)
			c.Header("Content-Type", "application/octet-stream")
			c.Header("Content-Disposition", "attachment; filename="+fileName)
			f.Write(c.Writer)
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate Excel"})
		return
	}

	// For action "validate", we redirect to the view page
	logger.Info("Flows validated and saved", "reference", reference, "count", len(flowsToExport))
	c.Redirect(http.StatusSeeOther, "/?tab=view")
}

func parsePorts(s string) []int {
	var result []int
	parts := strings.Split(s, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if strings.Contains(p, "-") {
			rangeParts := strings.Split(p, "-")
			if len(rangeParts) == 2 {
				start, _ := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
				end, _ := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
				for i := start; i <= end; i++ {
					result = append(result, i)
				}
			}
		} else {
			val, err := strconv.Atoi(p)
			if err == nil {
				result = append(result, val)
			}
		}
	}
	return result
}

func ensureCI(fqdn, ip string) {
	var ci models.CI
	err := database.DB.Where("ip = ?", ip).First(&ci).Error
	if err != nil {
		logger.Debug("Auto-creating CI", "fqdn", fqdn, "ip", ip)
		database.DB.Create(&models.CI{FQDN: fqdn, IP: ip})
	} else if ci.FQDN == "" && fqdn != "" {
		logger.Debug("Updating existing CI FQDN", "ip", ip, "new_fqdn", fqdn)
		ci.FQDN = fqdn
		database.DB.Save(&ci)
	}
}
