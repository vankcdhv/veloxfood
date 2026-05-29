package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	App      AppConfig      `mapstructure:"app"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Kafka    KafkaConfig    `mapstructure:"kafka"`
	RabbitMQ RabbitMQConfig `mapstructure:"rabbitmq"`
	DTM      DTMConfig      `mapstructure:"dtm"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	OTP      OTPConfig      `mapstructure:"otp"`
	SMTP     SMTPConfig     `mapstructure:"smtp"`
	Outbox   OutboxConfig   `mapstructure:"outbox"`
}

type AppConfig struct {
	Env          string `mapstructure:"env"`
	Port         int    `mapstructure:"port"`
	GRPCPort     int    `mapstructure:"grpc_port"`
	BaseURL      string `mapstructure:"base_url"`
	CookieSecure bool   `mapstructure:"cookie_secure"`
	CookieDomain string `mapstructure:"cookie_domain"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Name     string `mapstructure:"name"`
	SSLMode  string `mapstructure:"ssl_mode"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type KafkaConfig struct {
	Brokers []string `mapstructure:"brokers"`
}

type RabbitMQConfig struct {
	URL string `mapstructure:"url"`
}

type DTMConfig struct {
	Addr string `mapstructure:"addr"`
}

// JWTConfig holds RSA key material and token TTLs.
// Path fields take precedence over PEM (base64) fields — use PEM for prod env vars.
type JWTConfig struct {
	PrivateKeyPath string        `mapstructure:"private_key_path"`
	PublicKeyPath  string        `mapstructure:"public_key_path"`
	PrivateKeyPEM  string        `mapstructure:"private_key_pem"`  // base64-encoded PEM, prod fallback
	PublicKeyPEM   string        `mapstructure:"public_key_pem"`   // base64-encoded PEM, prod fallback
	AccessTTL      time.Duration `mapstructure:"access_ttl"`
	RefreshTTL     time.Duration `mapstructure:"refresh_ttl"`
}

// OTPConfig controls OTP generation and verification.
type OTPConfig struct {
	TTL         time.Duration `mapstructure:"ttl"`
	MaxAttempts int           `mapstructure:"max_attempts"`
}

// OutboxConfig controls the transactional outbox dispatcher polling loop.
type OutboxConfig struct {
	TickEvery   time.Duration `mapstructure:"tick_every"`
	BatchSize   int           `mapstructure:"batch_size"`
	MaxAttempts int           `mapstructure:"max_attempts"`
}

// SMTPConfig for outbound email delivery.
type SMTPConfig struct {
	Host       string `mapstructure:"host"`
	Port       int    `mapstructure:"port"`
	Username   string `mapstructure:"username"`
	Password   string `mapstructure:"password"`
	From       string `mapstructure:"from"`
	TLSEnabled bool   `mapstructure:"tls_enabled"`
}

func Load(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.SetEnvPrefix("")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
