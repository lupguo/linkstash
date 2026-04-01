package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server     ServerConfig   `yaml:"server"`
	Auth       AuthConfig     `yaml:"auth"`
	Database   DatabaseConfig `yaml:"database"`
	Log        LogConfig      `yaml:"log"`
	LLM        LLMConfig      `yaml:"llm"`
	Short      ShortConfig    `yaml:"short"`
	Categories   []string           `yaml:"categories"`
	NetworkTypes []NetworkTypeOption `yaml:"network_types"`
	Proxy      ProxyConfig    `yaml:"proxy"`
	Fetcher    FetcherConfig  `yaml:"fetcher"`
}

// FetcherConfig holds configurable fetch strategy settings.
type FetcherConfig struct {
	Strategies []string        `yaml:"strategies"`
	HTTP       HTTPFetchConfig `yaml:"http"`
	Browser    BrowserConfig   `yaml:"browser"`
}

// HTTPFetchConfig holds HTTP fetch strategy settings.
type HTTPFetchConfig struct {
	TimeoutSec int    `yaml:"timeout_sec"`
	MaxContent int    `yaml:"max_content"`
	UserAgent  string `yaml:"user_agent"`
}

// BrowserConfig holds browser fetch strategy settings (Rod headless Chrome).
type BrowserConfig struct {
	Enabled    bool   `yaml:"enabled"`     // Include in strategy chain (default false)
	BinPath    string `yaml:"bin_path"`    // Chromium binary path (empty = auto download)
	Headless   *bool  `yaml:"headless"`    // Headless mode (default true)
	TimeoutSec int    `yaml:"timeout_sec"` // Page timeout in seconds (default 30)
	MaxContent int    `yaml:"max_content"` // Max content bytes (default 51200)
	Lifecycle  string `yaml:"lifecycle"`   // "on-demand" or "singleton" (default "on-demand")
}

// IsHeadless returns whether to run in headless mode (defaults to true).
func (b BrowserConfig) IsHeadless() bool {
	if b.Headless == nil {
		return true
	}
	return *b.Headless
}

// LogConfig holds logging configuration.
type LogConfig struct {
	Level  string `yaml:"level"`  // debug, info, warn, error (default: info)
	File   string `yaml:"file"`   // log file path (default: "" = stdout only)
	Format string `yaml:"format"` // text, json (default: text)
}

// ProxyConfig holds optional HTTP/SOCKS proxy settings.
type ProxyConfig struct {
	HTTPProxy string `yaml:"http_proxy"` // e.g. "http://127.0.0.1:8118" or "socks5h://127.0.0.1:1080"
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

func (s ServerConfig) Addr() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

type AuthConfig struct {
	SecretKey      string `yaml:"secret_key"`
	JWTSecret      string `yaml:"jwt_secret"`
	JWTExpireHours int    `yaml:"jwt_expire_hours"`
}

type DatabaseConfig struct {
	Driver string       `yaml:"driver"` // "sqlite" (default) or "mysql"
	SQLite SQLiteConfig `yaml:"sqlite"`
	MySQL  MySQLConfig  `yaml:"mysql"`

	// Legacy field: kept for backward compatibility, maps to SQLite.Path
	Path string `yaml:"path"`
}

// SQLiteConfig holds SQLite-specific settings.
type SQLiteConfig struct {
	Path string `yaml:"path"` // Database file path (default: "./data/linkstash.db")
}

// MySQLConfig holds MySQL-specific settings.
type MySQLConfig struct {
	User         string `yaml:"user"`
	Password     string `yaml:"password"`
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	DBName       string `yaml:"dbname"`
	Charset      string `yaml:"charset"`       // default: utf8mb4
	MaxOpenConns int    `yaml:"max_open_conns"` // default: 25
	MaxIdleConns int    `yaml:"max_idle_conns"` // default: 5
}

// DSN returns the MySQL DSN string for GORM.
func (m MySQLConfig) DSN() string {
	charset := m.Charset
	if charset == "" {
		charset = "utf8mb4"
	}
	port := m.Port
	if port == 0 {
		port = 3306
	}
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		m.User, m.Password, m.Host, port, m.DBName, charset)
}

// IsSQLite returns true if the configured driver is SQLite (or unset, the default).
func (d DatabaseConfig) IsSQLite() bool {
	return d.Driver == "" || d.Driver == "sqlite"
}

// IsMySQL returns true if the configured driver is MySQL.
func (d DatabaseConfig) IsMySQL() bool {
	return d.Driver == "mysql"
}

// GetSQLitePath returns the effective SQLite database file path.
func (d DatabaseConfig) GetSQLitePath() string {
	if d.SQLite.Path != "" {
		return d.SQLite.Path
	}
	if d.Path != "" {
		return d.Path
	}
	return "./data/linkstash.db"
}

// NetworkTypeOption represents a network access type for the UI.
type NetworkTypeOption struct {
	Key   string `yaml:"key"`
	Label string `yaml:"label"`
}

// ShortTTLOption represents a single TTL choice for the UI dropdown.
type ShortTTLOption struct {
	Label string `yaml:"label"` // Display text, e.g. "永久", "1 天"
	Value string `yaml:"value"` // TTL value, e.g. "", "1d", "7d"
}

// ShortConfig holds short link generation settings.
type ShortConfig struct {
	TTLOptions []ShortTTLOption `yaml:"ttl_options"`
}

type LLMConfig struct {
	Chat      LLMEndpointConfig `yaml:"chat"`
	Embedding LLMEndpointConfig `yaml:"embedding"`
	Prompts   map[string]string `yaml:"prompts"`
}

type LLMEndpointConfig struct {
	Provider   string `yaml:"provider"`
	Endpoint   string `yaml:"endpoint"`
	APIKey     string `yaml:"api_key"`
	Model      string `yaml:"model"`
	Dimensions int    `yaml:"dimensions"`
}

// envVarRegex matches ${VAR_NAME} patterns
var envVarRegex = regexp.MustCompile(`\$\{([^}]+)\}`)

// Load reads a YAML config file and resolves environment variable references.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	// Resolve environment variable references: ${VAR} or ${VAR:default}
	// If VAR is set and non-empty, use its value; otherwise use default (if provided).
	resolved := envVarRegex.ReplaceAllStringFunc(string(data), func(match string) string {
		inner := strings.TrimSuffix(strings.TrimPrefix(match, "${"), "}")
		varName, defaultVal, hasDefault := strings.Cut(inner, ":")
		if val := os.Getenv(varName); val != "" {
			return val
		}
		if hasDefault {
			return defaultVal
		}
		return match // keep original if env var not set
	})

	var cfg Config
	if err := yaml.Unmarshal([]byte(resolved), &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	// Set defaults
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Auth.JWTExpireHours == 0 {
		cfg.Auth.JWTExpireHours = 72
	}
	// Database defaults
	if cfg.Database.Driver == "" {
		cfg.Database.Driver = "sqlite"
	}
	if cfg.Database.IsSQLite() && cfg.Database.GetSQLitePath() == "" {
		cfg.Database.SQLite.Path = "./data/linkstash.db"
	}
	if cfg.Database.IsMySQL() {
		if cfg.Database.MySQL.Charset == "" {
			cfg.Database.MySQL.Charset = "utf8mb4"
		}
		if cfg.Database.MySQL.Port == 0 {
			cfg.Database.MySQL.Port = 3306
		}
		if cfg.Database.MySQL.MaxOpenConns == 0 {
			cfg.Database.MySQL.MaxOpenConns = 25
		}
		if cfg.Database.MySQL.MaxIdleConns == 0 {
			cfg.Database.MySQL.MaxIdleConns = 5
		}
	}
	if cfg.Log.Level == "" {
		cfg.Log.Level = "info"
	}
	if cfg.Log.Format == "" {
		cfg.Log.Format = "text"
	}

	// Fetcher defaults
	if len(cfg.Fetcher.Strategies) == 0 {
		if cfg.Fetcher.Browser.Enabled {
			cfg.Fetcher.Strategies = []string{"http", "browser"}
		} else {
			cfg.Fetcher.Strategies = []string{"http"}
		}
	}
	if cfg.Fetcher.HTTP.TimeoutSec == 0 {
		cfg.Fetcher.HTTP.TimeoutSec = 15
	}
	if cfg.Fetcher.HTTP.MaxContent == 0 {
		cfg.Fetcher.HTTP.MaxContent = 51200
	}
	if cfg.Fetcher.HTTP.UserAgent == "" {
		cfg.Fetcher.HTTP.UserAgent = "LinkStash/1.0 (+https://github.com/lupguo/linkstash)"
	}
	if cfg.Fetcher.Browser.TimeoutSec == 0 {
		cfg.Fetcher.Browser.TimeoutSec = 30
	}
	if cfg.Fetcher.Browser.MaxContent == 0 {
		cfg.Fetcher.Browser.MaxContent = 51200
	}
	if cfg.Fetcher.Browser.Lifecycle == "" {
		cfg.Fetcher.Browser.Lifecycle = "on-demand"
	}

	// Default TTL options if not configured
	if len(cfg.Short.TTLOptions) == 0 {
		cfg.Short.TTLOptions = []ShortTTLOption{
			{Label: "never", Value: ""},
			{Label: "1 day", Value: "1d"},
			{Label: "7 days", Value: "7d"},
			{Label: "30 days", Value: "30d"},
		}
	}

	// Default network types if not configured
	if len(cfg.NetworkTypes) == 0 {
		cfg.NetworkTypes = []NetworkTypeOption{
			{Key: "internal", Label: "内网"},
			{Key: "domestic", Label: "国内"},
			{Key: "overseas", Label: "海外"},
			{Key: "unknown", Label: "未知"},
		}
	}

	return &cfg, nil
}
