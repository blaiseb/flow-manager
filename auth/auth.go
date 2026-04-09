package auth

import (
	"flow-manager/config"
	"flow-manager/logger"
	"flow-manager/models"
	"net/http"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// AuthRequired checks if user is authenticated and has required role.
func AuthRequired(db *gorm.DB, minRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var user models.User
		var authenticated bool

		if config.Global.Auth.Type == "proxy" {
			// Mode SSO via Reverse Proxy
			authenticated, user = handleProxyAuth(db, c)
		} else {
			// Mode Local via Session
			authenticated, user = handleLocalAuth(db, c)
		}

		if !authenticated {
			if config.Global.Auth.Type == "proxy" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized via proxy"})
			} else {
				c.Redirect(http.StatusSeeOther, "/login")
			}
			c.Abort()
			return
		}

		// Role hierarchy check
		if !HasPermission(user.Role, minRole) {
			logger.Warn("Access denied: insufficient permissions", "user", user.Username, "role", user.Role, "required", minRole)
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			c.Abort()
			return
		}

		c.Set("user", user)
		c.Next()
	}
}

func handleLocalAuth(db *gorm.DB, c *gin.Context) (bool, models.User) {
	session := sessions.Default(c)
	userID := session.Get("user_id")
	if userID == nil {
		return false, models.User{}
	}

	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		return false, models.User{}
	}
	return true, user
}

func handleProxyAuth(db *gorm.DB, c *gin.Context) (bool, models.User) {
	headerUser := config.Global.Auth.Proxy.HeaderUser
	username := c.GetHeader(headerUser)
	if username == "" {
		return false, models.User{}
	}

	// Fetch or Create user dynamically
	var user models.User
	err := db.Where("username = ?", username).First(&user).Error
	
	// Determine role from headers if provided
	role := models.RoleViewer
	headerRole := config.Global.Auth.Proxy.HeaderRole
	if headerRole != "" {
		groups := c.GetHeader(headerRole)
		role = mapGroupsToRole(groups)
	}

	if err != nil {
		// New SSO user
		user = models.User{
			Username: username,
			Role:     role,
			Password: "SSO_EXTERNAL_USER", // Dummy password
		}
		db.Create(&user)
		logger.Info("New SSO user created", "username", username, "role", role)
	} else if user.Role != role && role != models.RoleViewer {
		// Update role if changed in SSO
		user.Role = role
		db.Save(&user)
		logger.Debug("SSO user role updated", "username", username, "new_role", role)
	}

	return true, user
}

func mapGroupsToRole(groups string) string {
	mappings := config.Global.Auth.Proxy.RoleMappings
	if len(mappings) == 0 {
		return models.RoleViewer
	}

	// Split groups (LemonLDAP usually uses ; or ,)
	groupList := strings.FieldsFunc(groups, func(r rune) bool {
		return r == ';' || r == ',' || r == ' '
	})

	// Find the highest role mapped
	highestRole := models.RoleViewer
	weights := map[string]int{
		models.RoleViewer:    1,
		models.RoleRequestor: 2,
		models.RoleActor:     3,
		models.RoleAdmin:     4,
	}

	for _, g := range groupList {
		if r, ok := mappings[strings.TrimSpace(g)]; ok {
			if weights[r] > weights[highestRole] {
				highestRole = r
			}
		}
	}
	return highestRole
}

// MapOIDCGroupsToRole is specifically for OIDC where groups can be a slice or a string.
func MapOIDCGroupsToRole(groups interface{}) string {
	var groupList []string

	logger.Debug("OIDC Raw groups received", "raw", groups)

	switch g := groups.(type) {
	case string:
		groupList = strings.FieldsFunc(g, func(r rune) bool {
			return r == ';' || r == ',' || r == ' '
		})
	case []interface{}:
		for _, item := range g {
			if s, ok := item.(string); ok {
				groupList = append(groupList, s)
			}
		}
	case []string:
		groupList = g
	}

	logger.Debug("OIDC Extracted group list", "groups", groupList)

	mappings := config.Global.Auth.OIDC.RoleMappings
	if len(mappings) == 0 {
		return models.RoleViewer
	}

	highestRole := models.RoleViewer
	weights := map[string]int{
		models.RoleViewer:    1,
		models.RoleRequestor: 2,
		models.RoleActor:     3,
		models.RoleAdmin:     4,
	}

	for _, g := range groupList {
		if r, ok := mappings[strings.TrimSpace(g)]; ok {
			if weights[r] > weights[highestRole] {
				highestRole = r
			}
		}
	}
	return highestRole
}

// HasPermission checks if the userRole meets the minRequiredRole.
func HasPermission(userRole, minRequiredRole string) bool {
	weights := map[string]int{
		models.RoleViewer:    1,
		models.RoleRequestor: 2,
		models.RoleActor:     3,
		models.RoleAdmin:     4,
	}
	return weights[userRole] >= weights[minRequiredRole]
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
