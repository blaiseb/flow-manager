package handlers

import (
	"flow-manager/database"
	"flow-manager/logger"
	"flow-manager/models"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type IPInfo struct {
	IP          string
	Hostname    string
	Description string
	HasCI       bool
	CID         uint
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

	// Optimized VLAN matching using pre-parsed CIDRs
	parsedVlans := database.PreParseSubnets(vlans)

	// Fetch users only if admin
	if currentUser.Role == models.RoleAdmin {
		if err := h.DB.Find(&users).Error; err != nil {
			logger.Error("Failed to fetch users", "error", err)
		}
	}

	for i := range cis {
		cis[i].Vlan = database.MatchVLANOptimized(cis[i].IP, parsedVlans)
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
		
		// 1. Collecter les IPs des CIs correspondant au nom/IP recherché
		var matchingIPs []string
		h.DB.Model(&models.CI{}).Where("hostname LIKE ? OR ip LIKE ?", searchTerm, searchTerm).Pluck("ip", &matchingIPs)

		// 2. Identifier les VLANs correspondants au nom ou au subnet recherché
		var matchingVlanSubnets []string
		for _, v := range vlans {
			if strings.Contains(strings.ToLower(v.VLAN), strings.ToLower(searchQuery)) || 
			   strings.Contains(v.Subnet, searchQuery) {
				matchingVlanSubnets = append(matchingVlanSubnets, v.Subnet)
				
				// 3. Pour chaque VLAN trouvé, ajouter les IPs de tous les hosts de ce VLAN
				for _, ci := range cis {
					if ci.Vlan != nil && ci.Vlan.ID == v.ID {
						matchingIPs = append(matchingIPs, ci.IP)
					}
				}
			}
		}

		// Construire la requête finale avec les IPs des CIs et les subnets des VLANs
		if len(matchingIPs) > 0 || len(matchingVlanSubnets) > 0 {
			// On cherche dans source/target soit le terme direct, soit les IPs trouvées, soit les subnets trouvés
			db = db.Where("source_ip LIKE ? OR target_ip LIKE ? OR comment LIKE ? OR reference LIKE ? OR source_ip IN ? OR target_ip IN ? OR source_ip IN ? OR target_ip IN ?", 
				searchTerm, searchTerm, searchTerm, searchTerm, matchingIPs, matchingIPs, matchingVlanSubnets, matchingVlanSubnets)
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
		flows[i].SourceVlan = database.MatchVLANOptimized(flows[i].SourceIP, parsedVlans)

		if flows[i].TargetCI != nil {
			flows[i].TargetHostname = flows[i].TargetCI.Hostname
		}
		flows[i].TargetVlan = database.MatchVLANOptimized(flows[i].TargetIP, parsedVlans)
	}

	// Handle Selected VLAN
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
						info.CID = ci.ID
					}
					vlanIPs = append(vlanIPs, info)
				}
			}
		}
	}

	// Handle Selected CI
	var selectedCI *models.CI
	var ciFlows []models.FlowRequest
	ciIDParam := c.Query("ci_id")
	if ciIDParam != "" {
		ciID, err := strconv.Atoi(ciIDParam)
		if err == nil {
			for i := range cis {
				if cis[i].ID == uint(ciID) {
					selectedCI = &cis[i]
					break
				}
			}
			if selectedCI != nil {
				// Fetch finished flows for this CI or its VLAN
				query := h.DB.Preload("SourceCI").Preload("TargetCI").Where("status = ?", "terminé")

				// Build filter: (Source=IP OR Target=IP) OR (Source=VlanSubnet OR Target=VlanSubnet)
				if selectedCI.Vlan != nil {
					query = query.Where("(source_ip = ? OR target_ip = ? OR source_ip = ? OR target_ip = ?)", 
						selectedCI.IP, selectedCI.IP, selectedCI.Vlan.Subnet, selectedCI.Vlan.Subnet)
				} else {
					query = query.Where("(source_ip = ? OR target_ip = ?)", selectedCI.IP, selectedCI.IP)
				}

				query.Order("created_at desc").Find(&ciFlows)

				// Enhance ciFlows with display names
				for i := range ciFlows {
					if ciFlows[i].SourceCI != nil {
						ciFlows[i].SourceHostname = ciFlows[i].SourceCI.Hostname
					}
					ciFlows[i].SourceVlan = database.MatchVLANOptimized(ciFlows[i].SourceIP, parsedVlans)
					if ciFlows[i].TargetCI != nil {
						ciFlows[i].TargetHostname = ciFlows[i].TargetCI.Hostname
					}
					ciFlows[i].TargetVlan = database.MatchVLANOptimized(ciFlows[i].TargetIP, parsedVlans)
				}
			}
		}
	}

	c.HTML(http.StatusOK, "index.html", gin.H{
		"_csrf":        c.GetString("_csrf"),
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
		"SelectedCI":   selectedCI,
		"CiFlows":      ciFlows,
	})
}
