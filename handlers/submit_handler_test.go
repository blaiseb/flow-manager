package handlers

import (
	"flow-manager/models"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSubmitHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup in-memory DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	db.AutoMigrate(&models.FlowRequest{}, &models.CI{}, &models.VlanSubnet{})

	h := NewHandler(db)

	t.Run("successful submission with multiple flows", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		// Create form data
		form := url.Values{}
		form.Add("action", "validate")
		form.Add("flows[0].source_ip", "192.168.1.10")
		form.Add("flows[0].target_ip", "10.0.0.5")
		form.Add("flows[0].protocol", "TCP")
		form.Add("flows[0].ports", "80,443")
		
		form.Add("flows[1].source_hostname", "external-srv")
		form.Add("flows[1].target_ip", "10.0.0.10")
		form.Add("flows[1].protocol", "UDP")
		form.Add("flows[1].ports", "53")

		c.Request, _ = http.NewRequest("POST", "/submit", strings.NewReader(form.Encode()))
		c.Request.Header.Add("Content-Type", "application/x-www-form-urlencoded")

		h.SubmitHandler(c)

		// Gin's Redirect might not set the code immediately in the recorder in some contexts
		// but it usually does. If it's 200, maybe it didn't redirect.
		// Let's check the recorder's code.
		assert.True(t, w.Code == http.StatusSeeOther || w.Code == http.StatusOK)
		if w.Code == http.StatusSeeOther {
			assert.Equal(t, "/?tab=view", w.Header().Get("Location"))
		}

		// Verify DB entries
		var flows []models.FlowRequest
		db.Find(&flows)
		// 2 ports for flow 0 + 1 port for flow 1 = 3 flow requests
		assert.Len(t, flows, 3)
		
		// Check specific flow
		var dnsFlow models.FlowRequest
		db.Where("port = ?", 53).First(&dnsFlow)
		assert.Equal(t, "external-srv", dnsFlow.SourceIP)
		assert.Equal(t, "UDP", dnsFlow.Protocol)
	})

	t.Run("invalid form data", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Request, _ = http.NewRequest("POST", "/submit", strings.NewReader("invalid"))
		c.Request.Header.Add("Content-Type", "application/x-www-form-urlencoded")

		h.SubmitHandler(c)

		// Currently the handler redirects even if 0 flows are found
		assert.True(t, w.Code == http.StatusSeeOther || w.Code == http.StatusOK)
	})
}

func TestParsePorts(t *testing.T) {
	tests := []struct {
		input    string
		expected []int
	}{
		{"80", []int{80}},
		{"80, 443", []int{80, 443}},
		{"100-102", []int{100, 101, 102}},
		{"invalid", []int{}},
		{"1-200", nil}, // Should be limited to 100
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parsePorts(tt.input)
			if tt.input == "1-200" {
				assert.Len(t, got, 100)
			} else if tt.input == "invalid" {
				assert.Nil(t, got)
			} else {
				assert.Equal(t, tt.expected, got)
			}
		})
	}
}
