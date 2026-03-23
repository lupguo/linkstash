package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Auth     AuthConfig     `yaml:"auth"`
	Database DatabaseConfig `yaml:"database"`
	LLM      LLMConfig      `yaml:"llm"`
	Short      ShortConfig    `yaml:"short"`
	Categories []string       `yaml:"categories"`
	Proxy      ProxyConfig    `yaml:"proxy"`
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
	Path string `yaml:"path"`
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

	// Resolve environment variable references like ${VAR_NAME}
	resolved := envVarRegex.ReplaceAllStringFunc(string(data), func(match string) string {
		varName := strings.TrimSuffix(strings.TrimPrefix(match, "${"), "}")
		if val, ok := os.LookupEnv(varName); ok {
			return val
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
	if cfg.Database.Path == "" {
		cfg.Database.Path = "./data/linkstash.db"
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

	return &cfg, nil
}
