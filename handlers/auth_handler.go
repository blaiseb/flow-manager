package handlers

import (
	"context"
	"flow-manager/auth"
	"flow-manager/config"
	"flow-manager/database"
	"flow-manager/logger"
	"flow-manager/models"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

var (
	oidcProvider *oidc.Provider
	oidcConfig   oauth2.Config
	oidcVerifier *oidc.IDTokenVerifier
)

// InitOIDC initializes the OIDC provider and configuration.
func InitOIDC() {
	if config.Global.Auth.Type != "oidc" {
		return
	}

	ctx := context.Background()
	provider, err := oidc.NewProvider(ctx, config.Global.Auth.OIDC.Issuer)
	if err != nil {
		logger.Fatal("Failed to get OIDC provider", "error", err)
	}

	oidcProvider = provider
	oidcConfig = oauth2.Config{
		ClientID:     config.Global.Auth.OIDC.ClientID,
		ClientSecret: config.Global.Auth.OIDC.ClientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  config.Global.Auth.OIDC.RedirectURL,
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}

	oidcVerifier = provider.Verifier(&oidc.Config{ClientID: config.Global.Auth.OIDC.ClientID})
	logger.Info("OIDC initialized", "issuer", config.Global.Auth.OIDC.Issuer)
}

// ShowLogin displays the login page or redirects to OIDC.
func ShowLogin(c *gin.Context) {
	if config.Global.Auth.Type == "oidc" {
		state := "somestate" // In production, generate a random state and store it in session
		session := sessions.Default(c)
		session.Set("oidc_state", state)
		session.Save()
		c.Redirect(http.StatusFound, oidcConfig.AuthCodeURL(state))
		return
	}
	c.HTML(http.StatusOK, "login.html", gin.H{})
}

// OIDCCallback handles the callback from the OIDC provider.
func OIDCCallback(c *gin.Context) {
	session := sessions.Default(c)
	state := session.Get("oidc_state")
	if state == nil || c.Query("state") != state.(string) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid state"})
		return
	}

	ctx := context.Background()
	oauth2Token, err := oidcConfig.Exchange(ctx, c.Query("code"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to exchange token: " + err.Error()})
		return
	}

	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No id_token field in oauth2 token"})
		return
	}

	idToken, err := oidcVerifier.Verify(ctx, rawIDToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify ID Token: " + err.Error()})
		return
	}

	// Re-parse with dynamic groups claim
	var allClaims map[string]interface{}
	if err := idToken.Claims(&allClaims); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	username, _ := allClaims["preferred_username"].(string)
	if username == "" {
		username, _ = allClaims["email"].(string)
	}
	if username == "" {
		username, _ = allClaims["sub"].(string)
	}

	// Dynamic role matching from groups claim
	groupsClaimName := config.Global.Auth.OIDC.GroupsClaim
	role := models.RoleViewer
	if g, ok := allClaims[groupsClaimName]; ok {
		role = auth.MapOIDCGroupsToRole(g)
	}

	// Fetch or Create user
	var user models.User
	err = database.DB.Where("username = ?", username).First(&user).Error
	if err != nil {
		user = models.User{
			Username: username,
			Role:     role,
			Password: "OIDC_EXTERNAL_USER",
		}
		database.DB.Create(&user)
		logger.Info("New OIDC user created", "username", username, "role", role)
	} else {
		// Update role if changed
		if user.Role != role && role != models.RoleViewer {
			user.Role = role
			database.DB.Save(&user)
			logger.Debug("OIDC user role updated", "username", username, "new_role", role)
		}
	}

	session.Set("user_id", user.ID)
	session.Delete("oidc_state")
	session.Save()

	c.Redirect(http.StatusSeeOther, "/")
}

// Login handles the local login request.
func Login(c *gin.Context) {
	if config.Global.Auth.Type == "oidc" {
		c.Redirect(http.StatusFound, "/login")
		return
	}
	username := c.PostForm("username")
	password := c.PostForm("password")

	var user models.User
	if err := database.DB.Where("username = ?", username).First(&user).Error; err != nil {
		logger.Warn("Login failed: user not found", "username", username)
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{"error": "Utilisateur ou mot de passe incorrect"})
		return
	}

	if !auth.CheckPasswordHash(password, user.Password) {
		logger.Warn("Login failed: wrong password", "username", username)
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{"error": "Utilisateur ou mot de passe incorrect"})
		return
	}

	session := sessions.Default(c)
	session.Set("user_id", user.ID)
	session.Save()

	logger.Info("User logged in", "username", username, "role", user.Role)
	c.Redirect(http.StatusSeeOther, "/")
}

// Logout handles the logout request.
func Logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Save()
	if config.Global.Auth.Type == "oidc" {
		// In a real OIDC app, you might want to redirect to provider logout URL
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}
	c.Redirect(http.StatusSeeOther, "/login")
}

// CreateUser handles the creation of a new user.
func CreateUser(c *gin.Context) {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hashed, _ := auth.HashPassword(input.Password)
	user := models.User{
		Username: input.Username,
		Password: hashed,
		Role:     input.Role,
	}

	if err := database.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// UpdateUser handles user updates.
func UpdateUser(c *gin.Context) {
	id := c.Param("id")
	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user.Username = input.Username
	user.Role = input.Role
	if input.Password != "" {
		hashed, _ := auth.HashPassword(input.Password)
		user.Password = hashed
	}

	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// DeleteUser handles user deletion.
func DeleteUser(c *gin.Context) {
	id := c.Param("id")
	if err := database.DB.Delete(&models.User{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User deleted"})
}
