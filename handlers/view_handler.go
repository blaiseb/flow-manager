package handlers

import (
	"net/http"
	"strconv"

	"flow-manager/database"
	"flow-manager/models"

	"github.com/gin-gonic/gin"
)

type IPInfo struct {
	IP          string
	FQDN        string
	Description string
	HasCI       bool
}

func ViewHandler(c *gin.Context) {
	searchQuery := c.Query("search")
	var flows []models.FlowRequest
	var vlans []models.VlanSubnet
	var cis []models.CI
	activeTab := c.DefaultQuery("tab", "home")

	database.DB.Find(&vlans)
	database.DB.Find(&cis)

	// Map CIs for fast lookup
	ciMap := make(map[string]models.CI)
	for _, ci := range cis {
		ciMap[ci.IP] = ci
	}

	db := database.DB.Model(&models.FlowRequest{})
	if searchQuery != "" {
		activeTab = "view"
		searchTerm := "%" + searchQuery + "%"
		
		// 1. Find IPs of CIs matching the FQDN
		var matchingIPs []string
		database.DB.Model(&models.CI{}).Where("fqdn LIKE ? OR ip LIKE ?", searchTerm, searchTerm).Pluck("ip", &matchingIPs)

		// 2. Build the query: match by direct IP, by comment, OR by any IP found in step 1
		if len(matchingIPs) > 0 {
			db = db.Where("source_ip LIKE ? OR target_ip LIKE ? OR comment LIKE ? OR source_ip IN ? OR target_ip IN ?", 
				searchTerm, searchTerm, searchTerm, matchingIPs, matchingIPs)
		} else {
			db = db.Where("source_ip LIKE ? OR target_ip LIKE ? OR comment LIKE ?", 
				searchTerm, searchTerm, searchTerm)
		}
	}

	db.Order("created_at desc").Find(&flows)

	// Dynamically populate display fields for flows
	for i := range flows {
		if ci, ok := ciMap[flows[i].SourceIP]; ok {
			flows[i].SourceFQDN = ci.FQDN
		}
		flows[i].SourceVlan = database.MatchVLAN(flows[i].SourceIP, vlans)

		if ci, ok := ciMap[flows[i].TargetIP]; ok {
			flows[i].TargetFQDN = ci.FQDN
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
			ips, _ := database.GetIPsFromSubnet(selectedVlan.Subnet)
			for _, ipStr := range ips {
				info := IPInfo{IP: ipStr}
				if ci, ok := ciMap[ipStr]; ok {
					info.FQDN = ci.FQDN
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
		"SearchQuery":  searchQuery,
		"ActiveTab":    activeTab,
		"SelectedVlan": selectedVlan,
		"VlanIPs":      vlanIPs,
	})
}
