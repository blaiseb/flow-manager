package database

import (
	"fmt"
	"log"
	"net"
	"flow-manager/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// FindVLAN finds the corresponding VlanSubnet model for an IP address in the database.
func FindVLAN(ipStr string) (*models.VlanSubnet, error) {
	if ipStr == "" {
		return nil, nil // No IP, no VLAN.
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return nil, fmt.Errorf("invalid IP address format: %s", ipStr)
	}

	var subnets []models.VlanSubnet
	if err := DB.Find(&subnets).Error; err != nil {
		return nil, err
	}

	for _, s := range subnets {
		_, cidrNet, err := net.ParseCIDR(s.Subnet)
		if err != nil {
			log.Printf("Invalid CIDR in database: %s", s.Subnet)
			continue
		}
		if cidrNet.Contains(ip) {
			return &s, nil
		}
	}

	return nil, fmt.Errorf("no matching VLAN found")
}

// MatchVLAN finds the matching VLAN for an IP from a pre-fetched slice of subnets.
func MatchVLAN(ipStr string, subnets []models.VlanSubnet) *models.VlanSubnet {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return nil
	}

	for _, s := range subnets {
		_, cidrNet, err := net.ParseCIDR(s.Subnet)
		if err != nil {
			continue
		}
		if cidrNet.Contains(ip) {
			return &s
		}
	}
	return nil
}

// GetIPsFromSubnet returns all IP addresses in a given CIDR subnet.
func GetIPsFromSubnet(cidr string) ([]string, error) {
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	var ips []string
	// start at the network address
	ip := make(net.IP, len(ipnet.IP))
	copy(ip, ipnet.IP)

	for ipnet.Contains(ip) {
		ips = append(ips, ip.String())
		inc(ip)
	}

	// remove network and broadcast addresses for IPv4 if it's at least a /30
	ones, bits := ipnet.Mask.Size()
	if bits == 32 && ones <= 30 && len(ips) >= 4 {
		return ips[1 : len(ips)-1], nil
	}

	return ips, nil
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// InitDatabase initialise la connexion à la base de données et migre les schémas.
func InitDatabase() {
	var err error
	DB, err = gorm.Open(sqlite.Open("flows.db"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info), // Active le logging SQL
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Migration automatique des modèles pour créer/mettre à jour les tables.
	err = DB.AutoMigrate(&models.FlowRequest{}, &models.VlanSubnet{}, &models.CI{})
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Insérer des données de test pour les VLANs si la table est vide.
	var count int64
	DB.Model(&models.VlanSubnet{}).Count(&count)
	if count == 0 {
		log.Println("Seeding VlanSubnet table with initial data...")
		vlans := []models.VlanSubnet{
			{Subnet: "192.168.1.0/24", VLAN: "VLAN_SERVERS"},
			{Subnet: "10.0.0.0/8", VLAN: "VLAN_CORP"},
			{Subnet: "172.16.0.0/12", VLAN: "VLAN_GUEST"},
			{Subnet: "::1/128", VLAN: "VLAN_LOCALHOST"},
		}
		if err := DB.Create(&vlans).Error; err != nil {
			log.Fatalf("Failed to seed VlanSubnet table: %v", err)
		}
	}

	log.Println("Database connection successful and schemas migrated.")
}
