package database

import (
	"flow-manager/models"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestFindVLAN(t *testing.T) {
	// Setup in-memory SQLite for testing
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&models.VlanSubnet{})
	require.NoError(t, err)

	vlans := []models.VlanSubnet{
		{Subnet: "192.168.1.0/24", VLAN: "SERVERS"},
		{Subnet: "10.0.0.0/8", VLAN: "CORP"},
	}
	db.Create(&vlans)

	tests := []struct {
		name     string
		ip       string
		wantVlan string
		wantErr  bool
	}{
		{"match servers", "192.168.1.50", "SERVERS", false},
		{"match corp", "10.1.2.3", "CORP", false},
		{"no match", "172.16.0.1", "", true},
		{"invalid ip", "not-an-ip", "", true},
		{"empty ip", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FindVLAN(db, tt.ip)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.ip == "" {
					assert.Nil(t, got)
				} else {
					assert.NotNil(t, got)
					assert.Equal(t, tt.wantVlan, got.VLAN)
				}
			}
		})
	}
}

func TestPreParseSubnets(t *testing.T) {
	vlans := []models.VlanSubnet{
		{Subnet: "192.168.1.0/24", VLAN: "V1"},
		{Subnet: "invalid-cidr", VLAN: "V2"},
	}

	parsed := PreParseSubnets(vlans)
	assert.Len(t, parsed, 1)
	assert.Equal(t, "192.168.1.0/24", parsed[0].Net.String())
	assert.Equal(t, "V1", parsed[0].Source.VLAN)
}

func TestMatchVLANOptimized(t *testing.T) {
	_, n1, _ := net.ParseCIDR("192.168.1.0/24")
	_, n2, _ := net.ParseCIDR("10.0.0.0/8")
	
	parsed := []ParsedSubnet{
		{Net: n1, Source: &models.VlanSubnet{VLAN: "V1"}},
		{Net: n2, Source: &models.VlanSubnet{VLAN: "V2"}},
	}

	tests := []struct {
		name string
		ip   string
		want string
	}{
		{"match v1", "192.168.1.10", "V1"},
		{"match v2", "10.255.255.254", "V2"},
		{"no match", "8.8.8.8", ""},
		{"invalid ip", "invalid", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchVLANOptimized(tt.ip, parsed)
			if tt.want == "" {
				assert.Nil(t, got)
			} else {
				assert.NotNil(t, got)
				assert.Equal(t, tt.want, got.VLAN)
			}
		})
	}
}

func TestGetIPsFromSubnet(t *testing.T) {
	tests := []struct {
		name    string
		cidr    string
		count   int
		wantErr bool
	}{
		{"small subnet /30", "192.168.1.0/30", 2, false}, // 1.1, 1.2 (gw and broadcast excluded if it detects 32 bits bits)
		{"tiny subnet /32", "192.168.1.1/32", 1, false},
		{"security limit /15", "10.0.0.0/15", 0, true},
		{"allowed limit /16", "10.0.0.0/16", 65536, false},
		{"invalid cidr", "invalid", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ips, err := GetIPsFromSubnet(tt.cidr)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.count > 0 && tt.count < 100 { // Only check exact count for small subnets
					assert.Len(t, ips, tt.count)
				} else if tt.count >= 100 {
					assert.Greater(t, len(ips), 100)
				}
			}
		})
	}
}
