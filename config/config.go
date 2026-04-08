package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Port   int    `yaml:"port"`
		Secret string `yaml:"secret"`
	} `yaml:"server"`
	Log struct {
		Level string `yaml:"level"` // debug, info, warn, error
		Debug bool   `yaml:"debug"`
	} `yaml:"log"`
	Database struct {
		Type     string `yaml:"type"` // sqlite or postgres
		SQLite   struct {
			File string `yaml:"file"`
		} `yaml:"sqlite"`
		Postgres struct {
			Host     string `yaml:"host"`
			Port     int    `yaml:"port"`
			User     string `yaml:"user"`
			Password string `yaml:"password"`
			DBName   string `yaml:"dbname"`
			SSLMode  string `yaml:"sslmode"`
		} `yaml:"postgres"`
	} `yaml:"database"`
	Auth struct {
		Type   string `yaml:"type"` // local, proxy, or oidc
		Proxy  struct {
			HeaderUser   string `yaml:"header_user"`
			HeaderRole   string `yaml:"header_role"`
			RoleMappings map[string]string `yaml:"role_mappings"`
		} `yaml:"proxy"`
		OIDC struct {
			Issuer       string `yaml:"issuer"`
			ClientID     string `yaml:"client_id"`
			ClientSecret string `yaml:"client_secret"`
			RedirectURL  string `yaml:"redirect_url"`
			GroupsClaim  string `yaml:"groups_claim"` // Nom du claim pour les groupes, ex: "groups"
			RoleMappings map[string]string `yaml:"role_mappings"`
		} `yaml:"oidc"`
	} `yaml:"auth"`
}

var Global Config

func LoadConfig(path string) error {
	// Set defaults
	Global.Server.Port = 8080
	Global.Log.Level = "info"
	Global.Database.Type = "sqlite"
	Global.Database.SQLite.File = "flows.db"
	Global.Auth.Type = "local"
	Global.Auth.OIDC.GroupsClaim = "groups"

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&Global); err != nil {
		return err
	}

	if secret := os.Getenv("FLOW_SESSION_SECRET"); secret != "" {
		Global.Server.Secret = secret
	}

	return nil
}
