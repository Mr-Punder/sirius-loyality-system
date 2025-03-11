package config

import (
	"flag"
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type LogConfig struct {
	Level     string `yaml:"level"`
	Path      string `yaml:"path"`
	ErrorPath string `yaml:"errorpath"`
}

type ServerConfig struct {
	RunAddress string `yaml:"runaddress"`
}

// Config представляет структуру конфигурации
type Config struct {
	Server ServerConfig `yaml:"server"`

	Database struct {
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		DBName   string `yaml:"dbname"`
		SSLMode  string `yaml:"sslmode"`
	} `yaml:"database"`

	Auth struct {
		SecretToken string `yaml:"secret_token"`
	} `yaml:"auth"`

	Log LogConfig `yaml:"logger"`
}

// LoadConfig загружает конфигурацию из файла YAML
func LoadConfig() (*Config, error) {
	var filepath string
	flag.StringVar(&filepath, "c", "config.yaml", "config path")
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	config := &Config{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config data: %w", err)
	}

	return config, nil
}
