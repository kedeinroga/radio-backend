package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all application configuration
type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	Redis     RedisConfig
	JWT       JWTConfig
	External  ExternalConfig
	Logging   LoggingConfig
	Analytics AnalyticsConfig
	Security  SecurityConfig
	CORS      CORSConfig
	Cache     CacheConfig
	Features  FeatureFlags
	Vault     VaultConfig
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port    string
	Host    string
	Env     string
	Timeout time.Duration
	BaseURL string // NUEVO: URL base para generar links SEO
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	URL                string
	MaxConnections     int
	MaxIdleConnections int
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	URL      string
	Password string
	DB       int
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	Secret            string
	Expiration        time.Duration
	RefreshExpiration time.Duration
	PrivateKeyPath    string
	PublicKeyPath     string
}

// ExternalConfig holds external API configuration
type ExternalConfig struct {
	RadioBrowserAPIURL string
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string
	Format string
}

// AnalyticsConfig holds analytics configuration
type AnalyticsConfig struct {
	BatchSize     int
	FlushInterval time.Duration
}

// SecurityConfig holds security configuration
type SecurityConfig struct {
	BcryptCost      int
	RateLimitReqs   int
	RateLimitWindow time.Duration
}

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
}

// CacheConfig holds cache configuration
type CacheConfig struct {
	StationMaxAge     time.Duration
	SearchCacheTTL    time.Duration
	SyncInterval      time.Duration
	SyncBatchSize     int
	CleanupInterval   time.Duration
	InactiveThreshold time.Duration
}

// FeatureFlags holds feature flags
type FeatureFlags struct {
	PremiumContent   bool
	Analytics        bool
	VaultIntegration bool
}

// VaultConfig holds HashiCorp Vault configuration
type VaultConfig struct {
	Addr       string
	Token      string
	SecretPath string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if it exists (ignore error in production)
	_ = godotenv.Load()

	cfg := &Config{
		Server: ServerConfig{
			Port:    getEnv("SERVER_PORT", "8080"),
			Host:    getEnv("SERVER_HOST", "0.0.0.0"),
			Env:     getEnv("SERVER_ENV", "development"),
			Timeout: getDurationEnv("SERVER_TIMEOUT", 30*time.Second),
			BaseURL: getEnv("SERVER_BASE_URL", "http://localhost:8080"), // NUEVO: URL base
		},
		Database: DatabaseConfig{
			URL:                getEnv("DATABASE_URL", ""),
			MaxConnections:     getIntEnv("DATABASE_MAX_CONNECTIONS", 25),
			MaxIdleConnections: getIntEnv("DATABASE_MAX_IDLE_CONNECTIONS", 5),
		},
		Redis: RedisConfig{
			URL:      getEnv("REDIS_URL", "redis://localhost:6379/0"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getIntEnv("REDIS_DB", 0),
		},
		JWT: JWTConfig{
			Secret:            getEnv("JWT_SECRET", ""),
			Expiration:        getDurationEnv("JWT_EXPIRATION", 24*time.Hour),
			RefreshExpiration: getDurationEnv("JWT_REFRESH_EXPIRATION", 168*time.Hour),
			PrivateKeyPath:    getEnv("JWT_PRIVATE_KEY_PATH", "./keys/jwt-private.pem"),
			PublicKeyPath:     getEnv("JWT_PUBLIC_KEY_PATH", "./keys/jwt-public.pem"),
		},
		External: ExternalConfig{
			RadioBrowserAPIURL: getEnv("RADIO_BROWSER_API_URL", "https://all.api.radio-browser.info"),
		},
		Logging: LoggingConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
		Analytics: AnalyticsConfig{
			BatchSize:     getIntEnv("ANALYTICS_BATCH_SIZE", 100),
			FlushInterval: getDurationEnv("ANALYTICS_FLUSH_INTERVAL", 10*time.Second),
		},
		Security: SecurityConfig{
			BcryptCost:      getIntEnv("BCRYPT_COST", 12),
			RateLimitReqs:   getIntEnv("RATE_LIMIT_REQUESTS", 100),
			RateLimitWindow: getDurationEnv("RATE_LIMIT_WINDOW", 1*time.Minute),
		},
		CORS: CORSConfig{
			AllowedOrigins: getSliceEnv("CORS_ALLOWED_ORIGINS", []string{"*"}),
			AllowedMethods: getSliceEnv("CORS_ALLOWED_METHODS", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
			AllowedHeaders: getSliceEnv("CORS_ALLOWED_HEADERS", []string{"Content-Type", "Authorization", "X-Request-ID"}),
		},
		Cache: CacheConfig{
			StationMaxAge:     getDurationEnv("CACHE_STATION_MAX_AGE", 7*24*time.Hour),
			SearchCacheTTL:    getDurationEnv("CACHE_SEARCH_TTL", 1*time.Hour),
			SyncInterval:      getDurationEnv("CACHE_SYNC_INTERVAL", 1*time.Hour),
			SyncBatchSize:     getIntEnv("CACHE_SYNC_BATCH_SIZE", 100),
			CleanupInterval:   getDurationEnv("CACHE_CLEANUP_INTERVAL", 24*time.Hour),
			InactiveThreshold: getDurationEnv("CACHE_INACTIVE_THRESHOLD", 30*24*time.Hour),
		},
		Features: FeatureFlags{
			PremiumContent:   getBoolEnv("FEATURE_PREMIUM_CONTENT", true),
			Analytics:        getBoolEnv("FEATURE_ANALYTICS", true),
			VaultIntegration: getBoolEnv("FEATURE_VAULT_INTEGRATION", false),
		},
		Vault: VaultConfig{
			Addr:       getEnv("VAULT_ADDR", ""),
			Token:      getEnv("VAULT_TOKEN", ""),
			SecretPath: getEnv("VAULT_SECRET_PATH", "secret/radio-backend"),
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Database.URL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}

	if c.JWT.Secret == "" && c.JWT.PrivateKeyPath == "" {
		return fmt.Errorf("either JWT_SECRET or JWT_PRIVATE_KEY_PATH is required")
	}

	if c.Server.Env != "development" && c.Server.Env != "production" && c.Server.Env != "staging" {
		return fmt.Errorf("SERVER_ENV must be development, staging, or production")
	}

	return nil
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	return c.Server.Env == "development"
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	return c.Server.Env == "production"
}

// Helper functions

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getSliceEnv(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}
