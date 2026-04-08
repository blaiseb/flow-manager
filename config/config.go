package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Port int `yaml:"port"`
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
}

var Global Config

func LoadConfig(path string) error {
	// Set defaults
	Global.Server.Port = 8080
	Global.Log.Level = "info"
	Global.Database.Type = "sqlite"
	Global.Database.SQLite.File = "flows.db"

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Use defaults if file doesn't exist
		}
		return err
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&Global); err != nil {
		return err
	}

	return nil
}
