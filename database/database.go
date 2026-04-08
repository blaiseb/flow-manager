package database

import (
	"fmt"
	"net"
	"flow-manager/config"
	"flow-manager/models"
	"flow-manager/logger"
	"gorm.io/driver/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

var DB *gorm.DB

// FindVLAN finds the corresponding VlanSubnet model for an IP address in the database.
func FindVLAN(ipStr string) (*models.VlanSubnet, error) {
	if ipStr == "" {
		return nil, nil
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
			logger.Warn("Invalid CIDR in database", "subnet", s.Subnet)
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
	ip := make(net.IP, len(ipnet.IP))
	copy(ip, ipnet.IP)

	for ipnet.Contains(ip) {
		ips = append(ips, ip.String())
		inc(ip)
	}

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
	var dialector gorm.Dialector

	cfg := config.Global.Database

	if cfg.Type == "postgres" {
		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s",
			cfg.Postgres.Host, cfg.Postgres.User, cfg.Postgres.Password, 
			cfg.Postgres.DBName, cfg.Postgres.Port, cfg.Postgres.SSLMode)
		dialector = postgres.Open(dsn)
		logger.Info("Connecting to PostgreSQL database", "host", cfg.Postgres.Host, "dbname", cfg.Postgres.DBName)
	} else {
		dialector = sqlite.Open(cfg.SQLite.File)
		logger.Info("Connecting to SQLite database", "file", cfg.SQLite.File)
	}
	
	dbLogLevel := gormlogger.Silent
	if config.Global.Log.Debug {
		dbLogLevel = gormlogger.Info
	} else {
		dbLogLevel = gormlogger.Error
	}

	DB, err = gorm.Open(dialector, &gorm.Config{
		Logger: gormlogger.Default.LogMode(dbLogLevel),
	})
	if err != nil {
		logger.Fatal("Failed to connect to database", "error", err)
	}

	err = DB.AutoMigrate(&models.FlowRequest{}, &models.VlanSubnet{}, &models.CI{}, &models.User{})
	if err != nil {
		logger.Fatal("Failed to migrate database", "error", err)
	}

	var count int64
	DB.Model(&models.VlanSubnet{}).Count(&count)
	if count == 0 {
		logger.Info("Seeding VlanSubnet table with initial data...")
		vlans := []models.VlanSubnet{
			{Subnet: "192.168.1.0/24", VLAN: "VLAN_SERVERS"},
			{Subnet: "10.0.0.0/8", VLAN: "VLAN_CORP"},
			{Subnet: "172.16.0.0/12", VLAN: "VLAN_GUEST"},
			{Subnet: "::1/128", VLAN: "VLAN_LOCALHOST"},
		}
		if err := DB.Create(&vlans).Error; err != nil {
			logger.Fatal("Failed to seed VlanSubnet table", "error", err)
		}
	}

	logger.Info("Database connection successful and schemas migrated.")
}
