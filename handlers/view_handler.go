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

// ViewHandler handles the display of the main dashboard.
func (h *Handler) ViewHandler(c *gin.Context) {
	searchQuery := c.Query("search")
	var flows []models.FlowRequest
	var vlans []models.VlanSubnet
	var cis []models.CI
	var references []string
	var users []models.User
	activeTab := c.DefaultQuery("tab", "home")

	// Current User
	var currentUser models.User
	if val, ok := c.Get("user"); ok {
		if u, ok := val.(models.User); ok {
			currentUser = u
		}
	}

	if err := h.DB.Find(&vlans).Error; err != nil {
		logger.Error("Failed to fetch VLANs", "error", err)
	}
	if err := h.DB.Find(&cis).Error; err != nil {
		logger.Error("Failed to fetch CIs", "error", err)
	}

	// Fetch users only if admin
	if currentUser.Role == models.RoleAdmin {
		if err := h.DB.Find(&users).Error; err != nil {
			logger.Error("Failed to fetch users", "error", err)
		}
	}

	for i := range cis {
		cis[i].Vlan = database.MatchVLAN(cis[i].IP, vlans)
	}
	
	h.DB.Model(&models.FlowRequest{}).Distinct().Pluck("reference", &references)

	ciMap := make(map[string]models.CI)
	for _, ci := range cis {
		ciMap[ci.IP] = ci
	}

	db := h.DB.Model(&models.FlowRequest{})
	if searchQuery != "" {
		activeTab = "view"
		searchTerm := "%" + searchQuery + "%"
		
		var matchingIPs []string
		h.DB.Model(&models.CI{}).Where("hostname LIKE ? OR ip LIKE ?", searchTerm, searchTerm).Pluck("ip", &matchingIPs)

		if len(matchingIPs) > 0 {
			db = db.Where("source_ip LIKE ? OR target_ip LIKE ? OR comment LIKE ? OR reference LIKE ? OR source_ip IN ? OR target_ip IN ?", 
				searchTerm, searchTerm, searchTerm, searchTerm, matchingIPs, matchingIPs)
		} else {
			db = db.Where("source_ip LIKE ? OR target_ip LIKE ? OR comment LIKE ? OR reference LIKE ?", 
				searchTerm, searchTerm, searchTerm, searchTerm)
		}
	}

	db.Preload("SourceCI").Preload("TargetCI").Order("created_at desc").Find(&flows)

	for i := range flows {
		if flows[i].SourceCI != nil {
			flows[i].SourceHostname = flows[i].SourceCI.Hostname
		}
		flows[i].SourceVlan = database.MatchVLAN(flows[i].SourceIP, vlans)

		if flows[i].TargetCI != nil {
			flows[i].TargetHostname = flows[i].TargetCI.Hostname
		}
		flows[i].TargetVlan = database.MatchVLAN(flows[i].TargetIP, vlans)
	}

	var selectedVlan *models.VlanSubnet
	var vlanIPs []IPInfo
	vlanIDParam := c.Query("vlan_id")
	if vlanIDParam != "" {
		vlanID, err := strconv.Atoi(vlanIDParam)
		if err != nil {
			logger.Warn("Invalid vlan_id in query", "vlan_id", vlanIDParam, "error", err)
		} else {
			for i := range vlans {
				if vlans[i].ID == uint(vlanID) {
					selectedVlan = &vlans[i]
					break
				}
			}
			if selectedVlan != nil {
				ips, err := database.GetIPsFromSubnet(selectedVlan.Subnet)
				if err != nil {
					logger.Error("Failed to get IPs from subnet", "subnet", selectedVlan.Subnet, "error", err)
				}
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
	}

	c.HTML(http.StatusOK, "index.html", gin.H{
		"User":         currentUser,
		"Users":        users,
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
