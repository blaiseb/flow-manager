package models

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

// CI represents a Configuration Item.
type CI struct {
	gorm.Model
	Hostname    string      `json:"hostname" gorm:"index;not null" binding:"required"`
	IP          string      `json:"ip" gorm:"unique;index;not null" binding:"required,ip"`
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
	SourceIP      string     `json:"source_ip" gorm:"index;not null" binding:"required"`
	TargetIP      string     `json:"target_ip" gorm:"index;not null" binding:"required"`
	Protocol      string     `json:"protocol" gorm:"index;not null" binding:"required,oneof=TCP UDP BOTH ICMP"`
	Port          int        `json:"port" gorm:"index;not null" binding:"min=0,max=65535"`
	TimeLimit     *time.Time `json:"time_limit"`
	Comment       string     `json:"comment"`
	Reference     string     `json:"reference" gorm:"index;not null"`
	RuleNumber    string     `json:"rule_number"`
	ImplementedAt *time.Time `json:"implemented_at"`
	Status        string     `json:"status" gorm:"default:'demandé';not null"`

	// Associations SQL
	SourceCI *CI `json:"-" gorm:"foreignKey:SourceIP;references:IP"`
	TargetCI *CI `json:"-" gorm:"foreignKey:TargetIP;references:IP"`

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
	Username string `json:"username" gorm:"unique;index;not null" binding:"required,min=3"`
	Password string `json:"-" gorm:"not null"` // Hashé
	Role     string `json:"role" gorm:"default:'viewer';not null" binding:"required,oneof=viewer requestor actor admin"`
}

func (fr *FlowRequest) BeforeUpdate(tx *gorm.DB) (err error) {
	if fr.Status == "terminé" {
		now := time.Now()
		fr.ImplementedAt = &now
	}
	return
}

// VlanSubnet represents a VLAN / Subnet.
// StandardFlow represents a pre-defined flow type (e.g., HTTPS -> TCP/443).
type StandardFlow struct {
	gorm.Model
	Name     string `json:"name" gorm:"unique;not null" binding: "required"`
	Protocol string `json:"protocol" gorm:"not null" binding: "required"`
	Ports    string `json:"ports"`
}

type VlanSubnet struct {
	gorm.Model
	Subnet     string `json:"subnet" gorm:"unique;not null" binding:"required"`
	VLAN       string `json:"vlan" gorm:"not null" binding:"required"`
	Gateway    string `json:"gateway"`
	DNSServers string `json:"dns_servers"`
}
