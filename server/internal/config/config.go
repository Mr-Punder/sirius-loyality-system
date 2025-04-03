package config

import (
	"flag"
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

type LogConfig struct {
	Level     string `yaml:"level"`
	Path      string `yaml:"path"`
	ErrorPath string `yaml:"errorpath"`
	// Параметры ротации логов
	MaxSize    int  `yaml:"maxsize"`    // Максимальный размер файла в МБ перед ротацией
	MaxBackups int  `yaml:"maxbackups"` // Максимальное количество старых файлов логов для хранения
	MaxAge     int  `yaml:"maxage"`     // Максимальное количество дней для хранения старых файлов логов
	Compress   bool `yaml:"compress"`   // Сжимать ротированные файлы логов
}

type ServerConfig struct {
	RunAddress string `yaml:"runaddress"`
}

type StorageConfig struct {
	Type     string `yaml:"type"`     // Тип хранилища: "file" или "sqlite" (в будущем)
	DataPath string `yaml:"datapath"` // Путь к директории для хранения данных
}

type AdminConfig struct {
	JWTSecret string `yaml:"jwt_secret"` // Секретный ключ для JWT-токенов
}

type APIConfig struct {
	Token string `yaml:"token"` // Токен для API-запросов
}

// Config представляет структуру конфигурации
type Config struct {
	Server  ServerConfig  `yaml:"server"`
	Log     LogConfig     `yaml:"logger"`
	Storage StorageConfig `yaml:"storage"`
	Admin   AdminConfig   `yaml:"admin"`
	API     APIConfig     `yaml:"api"`
}

// LoadConfig загружает конфигурацию из файла YAML и учитывает флаги командной строки
func LoadConfig() (*Config, error) {
	// Определение флагов
	var logLevel string
	var runAddress string
	var logPath string
	var storageType string
	var storageDataPath string

	flag.StringVar(&logLevel, "log_level", "", "override log level")
	flag.StringVar(&runAddress, "run_address", "", "override run address")
	flag.StringVar(&logPath, "log_path", "", "override log path")
	flag.StringVar(&storageType, "storage_type", "", "override storage type (file or sqlite)")
	flag.StringVar(&storageDataPath, "storage_data_path", "", "override storage data path")

	// Получаем путь к конфигурационному файлу из переменной окружения или используем значение по умолчанию
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "cmd/loyalityserver/config.yaml"
	}

	// Чтение файла конфигурации
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	config := &Config{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config data: %w", err)
	}

	// Переопределение значений из флагов, если они установлены
	if logLevel != "" {
		config.Log.Level = logLevel
	}

	if runAddress != "" {
		config.Server.RunAddress = runAddress
	}

	if logPath != "" {
		config.Log.Path = logPath
	}

	if storageType != "" {
		config.Storage.Type = storageType
	}

	if storageDataPath != "" {
		config.Storage.DataPath = storageDataPath
	}

	// Установка значений по умолчанию, если они не заданы
	if config.Storage.Type == "" {
		config.Storage.Type = "file"
	}

	if config.Storage.DataPath == "" {
		config.Storage.DataPath = "./data"
	}

	// Установка значений по умолчанию для ротации логов
	if config.Log.MaxSize == 0 {
		config.Log.MaxSize = 10 // 10 МБ по умолчанию
	}

	if config.Log.MaxBackups == 0 {
		config.Log.MaxBackups = 5 // 5 файлов бэкапа по умолчанию
	}

	if config.Log.MaxAge == 0 {
		config.Log.MaxAge = 30 // 30 дней по умолчанию
	}

	return config, nil
}
