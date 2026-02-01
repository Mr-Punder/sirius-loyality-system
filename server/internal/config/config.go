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
	Type             string `yaml:"type"`              // Тип хранилища: "file", "postgres", "sqlite"
	DataPath         string `yaml:"datapath"`          // Путь к директории для хранения данных (для file)
	ConnectionString string `yaml:"connection_string"` // Строка подключения к PostgreSQL
	MigrationsPath   string `yaml:"migrations_path"`   // Путь к миграциям PostgreSQL/SQLite
	DBPath           string `yaml:"db_path"`           // Путь к файлу базы данных SQLite
}

type AdminConfig struct {
	JWTSecret  string `yaml:"jwt_secret"`  // Секретный ключ для JWT-токенов
	StaticDir  string `yaml:"static_dir"`  // Путь к директории со статическими файлами админки
	AdminsPath string `yaml:"admins_path"` // Путь к файлу со списком администраторов
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
func LoadConfig(config_path string) (*Config, error) {
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
	if config_path != "" {
		configPath = config_path
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

	// Переопределение значений из переменных окружения, если они установлены
	if envMigrationsPath := os.Getenv("MIGRATIONS_PATH"); envMigrationsPath != "" {
		config.Storage.MigrationsPath = envMigrationsPath
	}

	if envDBPath := os.Getenv("DB_PATH"); envDBPath != "" {
		config.Storage.DBPath = envDBPath
	}

	if envStaticDir := os.Getenv("ADMIN_STATIC_DIR"); envStaticDir != "" {
		config.Admin.StaticDir = envStaticDir
	}

	if envAdminsPath := os.Getenv("ADMIN_ADMINS_PATH"); envAdminsPath != "" {
		config.Admin.AdminsPath = envAdminsPath
	}

	if envConnStr := os.Getenv("POSTGRES_CONNECTION_STRING"); envConnStr != "" {
		config.Storage.ConnectionString = envConnStr
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

	// Установка значений по умолчанию для админки
	if config.Admin.StaticDir == "" {
		config.Admin.StaticDir = "./static/admin"
	}

	if config.Admin.AdminsPath == "" {
		config.Admin.AdminsPath = "./cmd/telegrambot/admin/admins.json"
	}

	return config, nil
}
