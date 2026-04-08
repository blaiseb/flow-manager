package models

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

// CI represents a Configuration Item.
type CI struct {
	gorm.Model
	Hostname    string      `json:"hostname" gorm:"index"`
	IP          string      `json:"ip" gorm:"unique;index"`
	Description string      `json:"description"`
	Vlan        *VlanSubnet `json:"vlan" gorm:"-"`
}

func (ci *CI) BeforeSave(tx *gorm.DB) (err error) {
	ci.Hostname = strings.ToLower(ci.Hostname)
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
	Reference     string     `json:"reference" gorm:"index"`
	RuleNumber    string     `json:"rule_number"`
	ImplementedAt *time.Time `json:"implemented_at"`
	Status        string     `json:"status" gorm:"default:'demandé'"`

	// Champs d'affichage dynamique (non stockés)
	SourceHostname string      `json:"source_hostname" gorm:"-"`
	SourceVlan     *VlanSubnet `json:"source_vlan" gorm:"-"`
	TargetHostname string      `json:"target_hostname" gorm:"-"`
	TargetVlan     *VlanSubnet `json:"target_vlan" gorm:"-"`
}

const (
	RoleViewer    = "viewer"
	RoleRequestor = "requestor"
	RoleActor     = "actor"
	RoleAdmin     = "admin"
)

type User struct {
	gorm.Model
	Username string `json:"username" gorm:"unique;index"`
	Password string `json:"-"` // Hashé
	Role     string `json:"role" gorm:"default:'viewer'"`
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
