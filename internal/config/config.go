package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server  ServerConfig  `yaml:"server"`
	Storage StorageConfig `yaml:"storage"`
	Routes  []RouteConfig `yaml:"routes"`
	Logging LoggingConfig `yaml:"logging"`
}

type ServerConfig struct {
	Ports    []int  `yaml:"ports"`
	Hostname string `yaml:"hostname"`
	TLS      TLSConfig `yaml:"tls"`
	RateLimit RateLimitConfig `yaml:"rate_limit"`
}

type TLSConfig struct {
	Enabled     bool              `yaml:"enabled"`
	CertFile    string            `yaml:"cert_file"`
	KeyFile     string            `yaml:"key_file"`
	LetsEncrypt LetsEncryptConfig `yaml:"letsencrypt"`
}

type LetsEncryptConfig struct {
	Enabled         bool     `yaml:"enabled"`
	Domains         []string `yaml:"domains"`
	Email           string   `yaml:"email"`
	CacheDir        string   `yaml:"cache_dir"`
	Staging         bool     `yaml:"staging"`
	HTTPPort        int      `yaml:"http_port"`
	RenewBeforeDays int      `yaml:"renew_before_days"`
}

type RateLimitConfig struct {
	Enabled    bool `yaml:"enabled"`
	MaxEmails  int  `yaml:"max_emails_per_minute"`
	MaxSize    int  `yaml:"max_email_size_mb"`
}

type StorageConfig struct {
	S3Compatible S3Config    `yaml:"s3_compatible"`
	Local        LocalConfig `yaml:"local"`
}

type S3Config struct {
	Enabled     bool   `yaml:"enabled"`
	Endpoint    string `yaml:"endpoint"`
	AccessKey   string `yaml:"access_key"`
	SecretKey   string `yaml:"secret_key"`
	Bucket      string `yaml:"bucket"`
	Region      string `yaml:"region"`
	UseSSL      bool   `yaml:"use_ssl"`
	PathPrefix  string `yaml:"path_prefix"`
}

type LocalConfig struct {
	Enabled   bool   `yaml:"enabled"`
	Directory string `yaml:"directory"`
}

type RouteConfig struct {
	Name        string     `yaml:"name"`
	Condition   Condition  `yaml:"condition"`
	Actions     []Action   `yaml:"actions"`
	Enabled     bool       `yaml:"enabled"`
}

type Condition struct {
	RecipientPattern string `yaml:"recipient_pattern"`
	SenderPattern    string `yaml:"sender_pattern"`
	SubjectPattern   string `yaml:"subject_pattern"`
}

type Action struct {
	Type     string            `yaml:"type"`
	Config   map[string]string `yaml:"config"`
	Enabled  bool              `yaml:"enabled"`
}

type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
	File   string `yaml:"file"`
}

func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

func validateConfig(config *Config) error {
	if len(config.Server.Ports) == 0 {
		return fmt.Errorf("at least one server port must be specified")
	}

	for _, port := range config.Server.Ports {
		if port < 1 || port > 65535 {
			return fmt.Errorf("invalid port number: %d", port)
		}
	}

	if config.Server.Hostname == "" {
		config.Server.Hostname = "localhost"
	}

	if !config.Storage.S3Compatible.Enabled && !config.Storage.Local.Enabled {
		return fmt.Errorf("at least one storage backend must be enabled")
	}

	if config.Storage.S3Compatible.Enabled {
		if config.Storage.S3Compatible.Endpoint == "" {
			return fmt.Errorf("S3 endpoint must be specified when S3 storage is enabled")
		}
		if config.Storage.S3Compatible.Bucket == "" {
			return fmt.Errorf("S3 bucket must be specified when S3 storage is enabled")
		}
	}

	if config.Storage.Local.Enabled && config.Storage.Local.Directory == "" {
		return fmt.Errorf("local directory must be specified when local storage is enabled")
	}

	for i, route := range config.Routes {
		if route.Name == "" {
			return fmt.Errorf("route %d must have a name", i)
		}
		if route.Condition.RecipientPattern == "" {
			return fmt.Errorf("route %s must have a recipient pattern", route.Name)
		}
		if len(route.Actions) == 0 {
			return fmt.Errorf("route %s must have at least one action", route.Name)
		}
	}

	return nil
}

func (c *Config) GetEnabledRoutes() []RouteConfig {
	var enabled []RouteConfig
	for _, route := range c.Routes {
		if route.Enabled {
			enabled = append(enabled, route)
		}
	}
	return enabled
}