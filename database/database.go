package database

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"flow-manager/config"
	"flow-manager/models"
	"flow-manager/logger"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// FindVLAN finds the corresponding VlanSubnet model for an IP address in the database.
func FindVLAN(db *gorm.DB, ipStr string) (*models.VlanSubnet, error) {
	if ipStr == "" {
		return nil, nil
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return nil, fmt.Errorf("invalid IP address format: %s", ipStr)
	}

	// Optimization for PostgreSQL: use the CIDR operator >>= (contained in or equals)
	if config.Global.Database.Type == "postgres" {
		var vlan models.VlanSubnet
		if err := db.Where("subnet >>= ?", ipStr).First(&vlan).Error; err == nil {
			return &vlan, nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	// Fallback for SQLite or if not found in Postgres (though >>= should work)
	var subnets []models.VlanSubnet
	if err := db.Find(&subnets).Error; err != nil {
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

// MatchVLAN finds the matching VLAN for an IP from a pre-fetched slice of subnets (non-optimized).
func MatchVLAN(ipStr string, subnets []models.VlanSubnet) *models.VlanSubnet {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return nil
	}

	for i := range subnets {
		_, cidrNet, err := net.ParseCIDR(subnets[i].Subnet)
		if err != nil {
			continue
		}
		if cidrNet.Contains(ip) {
			return &subnets[i]
		}
	}
	return nil
}

type ParsedSubnet struct {
	Net    *net.IPNet
	Source *models.VlanSubnet
}

// PreParseSubnets parses CIDR strings once to optimize matching.
func PreParseSubnets(subnets []models.VlanSubnet) []ParsedSubnet {
	result := make([]ParsedSubnet, 0, len(subnets))
	for i := range subnets {
		_, cidrNet, err := net.ParseCIDR(subnets[i].Subnet)
		if err != nil {
			logger.Warn("Invalid CIDR format during pre-parsing", "subnet", subnets[i].Subnet, "error", err)
			continue
		}
		result = append(result, ParsedSubnet{
			Net:    cidrNet,
			Source: &subnets[i],
		})
	}
	return result
}

// MatchVLANOptimized uses pre-parsed CIDRs to find a matching VLAN for an IP address.
func MatchVLANOptimized(ipStr string, parsed []ParsedSubnet) *models.VlanSubnet {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return nil
	}

	for _, p := range parsed {
		if p.Net.Contains(ip) {
			return p.Source
		}
	}
	return nil
}

// GetIPsFromSubnet returns all IP addresses in a given CIDR subnet.
// It limits enumeration to networks smaller than /16 to prevent OOM.
func GetIPsFromSubnet(cidr string) ([]string, error) {
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	ones, bits := ipnet.Mask.Size()
	// Security limit: refuse to enumerate IPs for large networks (e.g., > /16 for IPv4)
	if bits == 32 && ones < 16 {
		return nil, fmt.Errorf("subnet too large to enumerate (%s)", cidr)
	}
	if bits == 128 && ones < 112 {
		return nil, fmt.Errorf("IPv6 subnet too large to enumerate (%s)", cidr)
	}

	var ips []string
	ip := make(net.IP, len(ipnet.IP))
	copy(ip, ipnet.IP)

	for ipnet.Contains(ip) {
		ips = append(ips, ip.String())
		inc(ip)
	}

	ones, bits = ipnet.Mask.Size()
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

// SeedDefaultData ensures the database has an initial admin user.
func SeedDefaultData(db *gorm.DB) {
	// 1. Seed Admin User
	var admin models.User
	err := db.Where("username = ?", "admin").First(&admin).Error
	
	initialPassword := os.Getenv("INITIAL_ADMIN_PASSWORD")
	if initialPassword == "" {
		initialPassword = "admin"
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(initialPassword), 14)
	if err != nil {
		logger.Error("Failed to hash initial admin password", "error", err)
		return
	}

	if err == nil {
		// If password is still plain "admin" or not bcrypt, update it
		if admin.Password == "admin" || !strings.HasPrefix(admin.Password, "$2a$") {
			if os.Getenv("INITIAL_ADMIN_PASSWORD") == "" {
				logger.Warn("Admin password is still 'admin'. Please change it immediately.")
			}
			admin.Password = string(hashed)
			db.Save(&admin)
			logger.Info("Admin password updated to hashed version.")
		}
	} else {
		// Create admin
		newAdmin := models.User{
			Username: "admin",
			Password: string(hashed),
			Role:     models.RoleAdmin,
		}
		if err := db.Create(&newAdmin).Error; err != nil {
			logger.Error("Failed to create default admin user", "error", err)
		} else {
			if os.Getenv("INITIAL_ADMIN_PASSWORD") == "" {
				logger.Warn("Default admin 'admin' created with password 'admin'. Set INITIAL_ADMIN_PASSWORD next time.")
			} else {
				logger.Info("Default admin 'admin' created.")
			}
		}
	}
}

// InitDatabase initialise la connexion à la base de données et migre les schémas.
func InitDatabase() *gorm.DB {
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

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: gormlogger.Default.LogMode(dbLogLevel),
	})
	if err != nil {
		logger.Fatal("Failed to connect to database", "error", err)
	}

	err = db.AutoMigrate(&models.FlowRequest{}, &models.VlanSubnet{}, &models.CI{}, &models.User{})
	if err != nil {
		logger.Fatal("Failed to migrate database", "error", err)
	}

	logger.Info("Database connection successful and schemas migrated.")
	return db
}
