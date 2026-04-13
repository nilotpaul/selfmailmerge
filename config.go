package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

var presetNames []string
var presets = map[string]Config{
	"office365": {
		Host: "smtp.office365.com",
		Port: "587",
	},
	"google": {
		Host: "smtp.gmail.com",
		Port: "587",
	},
}

func init() {
	for k := range presets {
		presetNames = append(presetNames, k)
	}
}

type Config struct {
	IsTest bool

	Host       string
	Port       string
	User       string
	From       string
	Password   string
	PresetName string

	BodyContent    string
	SubjectContent string

	SpreadsheetPath string
	AttachmentPaths []string
}

func NewConfig() (*Config, error) {
	// via cli
	godotenv.Load()
	var (
		isTest   = flag.Bool("test", false, "test mode")
		user     = flag.String("username", "", "username for authentication")
		password = flag.String("password", "", "password for authentication")

		// templPath = flag.String("template", "template.html", "email template path") // TODO: fix
		spreadsheetPath = flag.String("csv", "", "spreadsheet file path")
		presetName      = flag.String("preset", "", strings.Join(presetNames, ","))
	)
	flag.Parse()

	cfg := &Config{
		IsTest:          *isTest,
		User:            *user,
		Password:        *password,
		SpreadsheetPath: *spreadsheetPath,
		PresetName:      *presetName,
	}
	// apply preset
	if preset, ok := presets[cfg.PresetName]; ok {
		cfg.Host = preset.Host
		cfg.Port = preset.Port
	}

	if err := loadFromEnv(cfg); err != nil {
		return nil, err
	}
	if len(cfg.User) == 0 {
		cfg.User = cfg.From
	}

	if err := validate(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func validate(cfg *Config) error {
	if len(cfg.Host) == 0 {
		return fmt.Errorf("smtp Host/Preset not provided")
	}
	if len(cfg.Port) == 0 {
		return fmt.Errorf("smtp Port/Preset not provided")
	}
	if len(cfg.User) == 0 {
		return fmt.Errorf("user for authentication not provided")
	}
	if len(cfg.Password) == 0 {
		return fmt.Errorf("password for authentication not provided")
	}
	if len(cfg.BodyContent) == 0 {
		return fmt.Errorf("email body not provided")
	}
	if len(cfg.SpreadsheetPath) == 0 {
		return fmt.Errorf("spreadsheet file path not provided")
	}

	return nil
}

func loadFromEnv(cfg *Config) error {
	if v := os.Getenv("SPREADSHEET_PATH"); len(v) > 0 {
		cfg.SpreadsheetPath = v
	}

	cfg.From = os.Getenv("FROM")
	cfg.SubjectContent = os.Getenv("SUBJECT")
	cfg.BodyContent = os.Getenv("BODY")
	cfg.AttachmentPaths = parseStringSlice(os.Getenv("ATTACHMENT"))

	return nil
}
