package config

import (
	"encoding/json"
	"os"
)

const configFileName = ".gatorconfig.json"

type Config struct {
	DBUrl       string `json:"db_url"`
	CurrentUser string `json:"current_user_name"`
}

func Read() (*Config, error) {
	filePath, err := getConfigFilePath()
	if err != nil {
		return nil, err
	}

	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(fileBytes, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) SetUser(userName string) error {
	c.CurrentUser = userName

	return write(*c)
}

func getConfigFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return homeDir + "/" + configFileName, nil
}

func write(cfg Config) error {
	filePath, err := getConfigFilePath()
	if err != nil {
		return err
	}

	cfgBytes, err := json.Marshal(&cfg)
	if err != nil {
		return err
	}

	err = os.WriteFile(filePath, cfgBytes, 0666)
	if err != nil {
		return err
	}

	return nil
}
