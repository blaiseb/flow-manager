package handlers

import (
	"flow-manager/database"
	"flow-manager/logger"
	"flow-manager/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type IPInfo struct {
	IP          string
	Hostname    string
	Description string
	HasCI       bool
}

func ViewHandler(c *gin.Context) {
	searchQuery := c.Query("search")
	var flows []models.FlowRequest
	var vlans []models.VlanSubnet
	var cis []models.CI
	var references []string
	activeTab := c.DefaultQuery("tab", "home")

	if err := database.DB.Find(&vlans).Error; err != nil {
		logger.Error("Failed to fetch VLANs", "error", err)
	}
	if err := database.DB.Find(&cis).Error; err != nil {
		logger.Error("Failed to fetch CIs", "error", err)
	}

	// Dynamically match VLAN for each CI in the management list
	for i := range cis {
		cis[i].Vlan = database.MatchVLAN(cis[i].IP, vlans)
	}
	
	database.DB.Model(&models.FlowRequest{}).Distinct().Pluck("reference", &references)

	// Map CIs for fast lookup
	ciMap := make(map[string]models.CI)
	for _, ci := range cis {
		ciMap[ci.IP] = ci
	}

	db := database.DB.Model(&models.FlowRequest{})
	if searchQuery != "" {
		activeTab = "view"
		searchTerm := "%" + searchQuery + "%"
		logger.Debug("Filtering flows", "query", searchQuery)
		
		// 1. Find IPs of CIs matching the Hostname
		var matchingIPs []string
		database.DB.Model(&models.CI{}).Where("hostname LIKE ? OR ip LIKE ?", searchTerm, searchTerm).Pluck("ip", &matchingIPs)

		// 2. Build the query including reference field
		if len(matchingIPs) > 0 {
			db = db.Where("source_ip LIKE ? OR target_ip LIKE ? OR comment LIKE ? OR reference LIKE ? OR source_ip IN ? OR target_ip IN ?", 
				searchTerm, searchTerm, searchTerm, searchTerm, matchingIPs, matchingIPs)
		} else {
			db = db.Where("source_ip LIKE ? OR target_ip LIKE ? OR comment LIKE ? OR reference LIKE ?", 
				searchTerm, searchTerm, searchTerm, searchTerm)
		}
	}

	db.Order("created_at desc").Find(&flows)

	// Dynamically populate display fields for flows
	for i := range flows {
		if ci, ok := ciMap[flows[i].SourceIP]; ok {
			flows[i].SourceHostname = ci.Hostname
		}
		flows[i].SourceVlan = database.MatchVLAN(flows[i].SourceIP, vlans)

		if ci, ok := ciMap[flows[i].TargetIP]; ok {
			flows[i].TargetHostname = ci.Hostname
		}
		flows[i].TargetVlan = database.MatchVLAN(flows[i].TargetIP, vlans)
	}

	// VLAN Detail Logic
	var selectedVlan *models.VlanSubnet
	var vlanIPs []IPInfo
	vlanIDParam := c.Query("vlan_id")
	if vlanIDParam != "" {
		vlanID, _ := strconv.Atoi(vlanIDParam)
		for i := range vlans {
			if vlans[i].ID == uint(vlanID) {
				selectedVlan = &vlans[i]
				break
			}
		}
		if selectedVlan != nil {
			logger.Debug("Loading VLAN detail", "vlan", selectedVlan.VLAN)
			ips, _ := database.GetIPsFromSubnet(selectedVlan.Subnet)
			for _, ipStr := range ips {
				info := IPInfo{IP: ipStr}
				if ci, ok := ciMap[ipStr]; ok {
					info.Hostname = ci.Hostname
					info.Description = ci.Description
					info.HasCI = true
				}
				vlanIPs = append(vlanIPs, info)
			}
		}
	}

	c.HTML(http.StatusOK, "index.html", gin.H{
		"Flows":        flows,
		"VLANs":        vlans,
		"CIs":          cis,
		"References":   references,
		"SearchQuery":  searchQuery,
		"ActiveTab":    activeTab,
		"SelectedVlan": selectedVlan,
		"VlanIPs":      vlanIPs,
	})
}
