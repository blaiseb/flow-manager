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
	SourceHostname string `form:"source_hostname"`
	SourceIP       string `form:"source_ip"`
	TargetHostname string `form:"target_hostname"`
	TargetIP       string `form:"target_ip"`
	Protocol       string `form:"protocol"`
	Ports          string `form:"ports"`
	TimeLimit      string `form:"time_limit"`
	Comment        string `form:"comment"`
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
		case "source_hostname": rows[idx].SourceHostname = val
		case "source_ip":       rows[idx].SourceIP = val
		case "target_hostname": rows[idx].TargetHostname = val
		case "target_ip":       rows[idx].TargetIP = val
		case "protocol":        rows[idx].Protocol = val
		case "ports":           rows[idx].Ports = val
		case "time_limit":      rows[idx].TimeLimit = val
		case "comment":         rows[idx].Comment = val
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
			source := sub.SourceIP
			if source == "" && sub.SourceHostname != "" {
				source = sub.SourceHostname // Cas externe
			}
			target := sub.TargetIP
			if target == "" && sub.TargetHostname != "" {
				target = sub.TargetHostname // Cas externe
			}

			flow := models.FlowRequest{
				SourceIP:   source,
				TargetIP:   target,
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
			
			// We need Hostnames for the Excel generation (even if not saved)
			flow.SourceHostname = sub.SourceHostname
			flow.TargetHostname = sub.TargetHostname
			
			flowsToExport = append(flowsToExport, flow)
		}
		
		if action == "validate" {
			if sub.SourceHostname != "" && sub.SourceIP != "" && !strings.Contains(sub.SourceIP, "/") {
				ensureCI(sub.SourceHostname, sub.SourceIP)
			}
			if sub.TargetHostname != "" && sub.TargetIP != "" && !strings.Contains(sub.TargetIP, "/") {
				ensureCI(sub.TargetHostname, sub.TargetIP)
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

func ensureCI(hostname, ip string) {
	var ci models.CI
	err := database.DB.Where("ip = ?", ip).First(&ci).Error
	if err != nil {
		logger.Debug("Auto-creating CI", "hostname", hostname, "ip", ip)
		database.DB.Create(&models.CI{Hostname: hostname, IP: ip})
	} else if ci.Hostname == "" && hostname != "" {
		logger.Debug("Updating existing CI Hostname", "ip", ip, "new_hostname", hostname)
		ci.Hostname = hostname
		database.DB.Save(&ci)
	}
}
