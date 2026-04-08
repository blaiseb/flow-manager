package handlers

import (
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

func (h *Handler) SubmitHandler(c *gin.Context) {
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
		case "source_hostname":
			rows[idx].SourceHostname = val
		case "source_ip":
			rows[idx].SourceIP = val
		case "target_hostname":
			rows[idx].TargetHostname = val
		case "target_ip":
			rows[idx].TargetIP = val
		case "protocol":
			rows[idx].Protocol = val
		case "ports":
			rows[idx].Ports = val
		case "time_limit":
			rows[idx].TimeLimit = val
		case "comment":
			rows[idx].Comment = val
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
				SourceIP:  source,
				TargetIP:  target,
				Protocol:  sub.Protocol,
				Port:      port,
				TimeLimit: timeLimit,
				Comment:   sub.Comment,
				Reference: reference,
				Status:    "demandé",
			}

			// We need Hostnames for the Excel generation (even if not saved)
			flow.SourceHostname = sub.SourceHostname
			flow.TargetHostname = sub.TargetHostname

			flowsToExport = append(flowsToExport, flow)
		}

		if action == "validate" {
			if sub.SourceHostname != "" && sub.SourceIP != "" && !strings.Contains(sub.SourceIP, "/") {
				h.ensureCI(sub.SourceHostname, sub.SourceIP)
			}
			if sub.TargetHostname != "" && sub.TargetIP != "" && !strings.Contains(sub.TargetIP, "/") {
				h.ensureCI(sub.TargetHostname, sub.TargetIP)
			}
		}
	}

	if action == "validate" && len(flowsToExport) > 0 {
		// Use batch insert for performance
		if err := h.DB.CreateInBatches(flowsToExport, 100).Error; err != nil {
			logger.Error("Failed to create flow requests in batch", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save flow requests"})
			return
		}
	}

	if action == "generate" {
		logger.Info("Generating Excel request (preview)", "count", len(flowsToExport))
		f, err := h.GenerateExcelFile(flowsToExport)
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

	if action == "markdown" {
		logger.Info("Generating Markdown request (preview)", "count", len(flowsToExport))
		md := h.GenerateMarkdown(flowsToExport)
		c.JSON(http.StatusOK, gin.H{"markdown": md})
		return
	}

	// For action "validate", we redirect to the view page
	logger.Info("Flows validated and saved", "reference", reference, "count", len(flowsToExport))
	c.Redirect(http.StatusSeeOther, "/?tab=view")
}

func (h *Handler) GenerateMarkdown(flows []models.FlowRequest) string {
	var sb strings.Builder
	sb.WriteString("bonjour, \nPouvez-vous réaliser les ouvertures de flux suivantes, \n\n")
	sb.WriteString("| Source | Destination | Protocole | Port | Commentaire |\n")
	sb.WriteString("| :--- | :--- | :--- | :--- | :--- |\n")

	for _, f := range flows {
		source := f.SourceIP
		if f.SourceHostname != "" {
			if f.SourceIP != "" {
				source = fmt.Sprintf("%s (%s)", f.SourceHostname, f.SourceIP)
			} else {
				source = f.SourceHostname
			}
		}
		target := f.TargetIP
		if f.TargetHostname != "" {
			if f.TargetIP != "" {
				target = fmt.Sprintf("%s (%s)", f.TargetHostname, f.TargetIP)
			} else {
				target = f.TargetHostname
			}
		}

		comment := f.Comment
		if f.TimeLimit != nil {
			if comment != "" {
				comment += " "
			}
			comment += fmt.Sprintf("(Jusqu'au %s)", f.TimeLimit.Format("2006-01-02"))
		}

		portStr := strconv.Itoa(f.Port)
		if f.Port == 0 {
			portStr = "Any"
		}

		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s |\n", source, target, f.Protocol, portStr, comment))
	}
	return sb.String()
}

func parsePorts(s string) []int {
	var result []int
	parts := strings.Split(s, ",")
	for _, p := range parts {
		if len(result) >= 100 {
			break
		}
		p = strings.TrimSpace(p)
		if strings.Contains(p, "-") {
			rangeParts := strings.Split(p, "-")
			if len(rangeParts) == 2 {
				start, err1 := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
				end, err2 := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
				if err1 == nil && err2 == nil {
					for i := start; i <= end; i++ {
						if len(result) >= 100 {
							break
						}
						result = append(result, i)
					}
				} else {
					logger.Warn("Invalid port range", "range", p)
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

func (h *Handler) ensureCI(hostname, ip string) {
	var ci models.CI
	err := h.DB.Where("ip = ?", ip).First(&ci).Error
	if err != nil {
		logger.Debug("Auto-creating CI", "hostname", hostname, "ip", ip)
		h.DB.Create(&models.CI{Hostname: hostname, IP: ip})
	} else if ci.Hostname == "" && hostname != "" {
		logger.Debug("Updating existing CI Hostname", "ip", ip, "new_hostname", hostname)
		ci.Hostname = hostname
		h.DB.Save(&ci)
	}
}
