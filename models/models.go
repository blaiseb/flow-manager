package models

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

// CI represents a Configuration Item.
type CI struct {
	gorm.Model
	FQDN        string      `json:"fqdn" gorm:"index"`
	IP          string      `json:"ip" gorm:"unique;index"`
	Description string      `json:"description"`
	Vlan        *VlanSubnet `json:"vlan" gorm:"-"`
}

func (ci *CI) BeforeSave(tx *gorm.DB) (err error) {
	ci.FQDN = strings.ToLower(ci.FQDN)
	return
}

// FlowRequest represents a unique flow request, associated by IP/Subnet only.
type FlowRequest struct {
	gorm.Model
	SourceIP      string     `json:"source_ip" gorm:"index"`
	TargetIP      string     `json:"target_ip" gorm:"index"`
	Protocol      string     `json:"protocol" gorm:"index"`
	Port          int        `json:"port" gorm:"index"`
	TimeLimit     *time.Time `json:"time_limit"`
	Comment       string     `json:"comment"`
	Reference     string     `json:"reference" gorm:"index"` // Nouveau champ de regroupement
	RuleNumber    string     `json:"rule_number"`
	ImplementedAt *time.Time `json:"implemented_at"`
	Status        string     `json:"status" gorm:"default:'demandé'"`

	// Champs d'affichage dynamique (non stockés)
	SourceFQDN string      `json:"source_fqdn" gorm:"-"`
	SourceVlan *VlanSubnet `json:"source_vlan" gorm:"-"`
	TargetFQDN string      `json:"target_fqdn" gorm:"-"`
	TargetVlan *VlanSubnet `json:"target_vlan" gorm:"-"`
}

func (fr *FlowRequest) BeforeUpdate(tx *gorm.DB) (err error) {
	if fr.Status == "terminé" {
		now := time.Now()
		fr.ImplementedAt = &now
	}
	return
}

// VlanSubnet represents a VLAN / Subnet.
type VlanSubnet struct {
	gorm.Model
	Subnet     string `json:"subnet" gorm:"unique"`
	VLAN       string `json:"vlan"`
	Gateway    string `json:"gateway"`
	DNSServers string `json:"dns_servers"`
}
