package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	App         AppConfig         `mapstructure:"app"`
	Database    DatabaseConfig    `mapstructure:"database"`
	Redis       RedisConfig       `mapstructure:"redis"`
	Kafka       KafkaConfig       `mapstructure:"kafka"`
	RabbitMQ    RabbitMQConfig    `mapstructure:"rabbitmq"`
	DTM         DTMConfig         `mapstructure:"dtm"`
	JWT         JWTConfig         `mapstructure:"jwt"`
	OTP         OTPConfig         `mapstructure:"otp"`
	SMTP        SMTPConfig        `mapstructure:"smtp"`
	Outbox      OutboxConfig      `mapstructure:"outbox"`
	GoogleOAuth GoogleOAuthConfig `mapstructure:"google_oauth"`
	MinIO       MinIOConfig       `mapstructure:"minio"`
	UserService UserServiceConfig `mapstructure:"user_service"`
	// LocationService addresses the location-service for cross-service gRPC
	// (e.g. store resolves a delivery room's building for shipping-fee lookup).
	LocationService ServiceEndpoint `mapstructure:"location_service"`
	// StoreService addresses the store-service for cross-service gRPC
	// (e.g. promotion verifies store ownership before mutating vouchers).
	StoreService ServiceEndpoint `mapstructure:"store_service"`
	// PromotionService addresses the promotion-service for cross-service gRPC
	// (e.g. order saga applies/releases voucher discounts).
	PromotionService ServiceEndpoint `mapstructure:"promotion_service"`
	// PaymentService addresses the payment-service for cross-service gRPC
	// (e.g. order saga captures or refunds payments).
	PaymentService ServiceEndpoint `mapstructure:"payment_service"`
	// OrderService addresses the order-service for cross-service gRPC
	// (e.g. review validates order completion/ownership via GetOrder).
	OrderService ServiceEndpoint `mapstructure:"order_service"`
	// MoMo holds credentials and endpoint for MoMo payment gateway.
	// Default values point to the public MoMo sandbox for development.
	MoMo MoMoConfig `mapstructure:"momo"`
	// Firebase holds the Admin SDK service-account path + project for the
	// notification service (realtime order status to Firestore + FCM push).
	Firebase FirebaseConfig `mapstructure:"firebase"`
}

// FirebaseConfig points the notification service at Firebase Admin SDK creds.
// If ServiceAccountPath is empty the realtime/FCM features degrade to no-op.
type FirebaseConfig struct {
	ServiceAccountPath string `mapstructure:"service_account_path"`
	ProjectID          string `mapstructure:"project_id"`
}

// MoMoConfig holds MoMo v2 payment gateway credentials.
// Switching demo↔prod requires only config changes (no code change).
type MoMoConfig struct {
	Endpoint    string `mapstructure:"endpoint"`
	PartnerCode string `mapstructure:"partner_code"`
	AccessKey   string `mapstructure:"access_key"`
	SecretKey   string `mapstructure:"secret_key"`
	RedirectURL string `mapstructure:"redirect_url"`
	IpnURL      string `mapstructure:"ipn_url"`
	Env         string `mapstructure:"env"` // "demo" | "prod"
}

// UserServiceConfig points other services at the user-service for cross-service
// calls (e.g. permission checks over gRPC).
type UserServiceConfig struct {
	GRPCAddr string `mapstructure:"grpc_addr"`
}

// ServiceEndpoint addresses a peer service for cross-service gRPC calls.
type ServiceEndpoint struct {
	GRPCAddr string `mapstructure:"grpc_addr"`
}

// GoogleOAuthConfig holds Google OAuth 2.0 web client credentials.
type GoogleOAuthConfig struct {
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
	RedirectURI  string `mapstructure:"redirect_uri"`
}

// MinIOConfig holds S3-compatible object storage settings.
type MinIOConfig struct {
	Endpoint  string `mapstructure:"endpoint"`
	AccessKey string `mapstructure:"access_key"`
	SecretKey string `mapstructure:"secret_key"`
	Bucket    string `mapstructure:"bucket"`
	UseSSL    bool   `mapstructure:"use_ssl"`
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
	PrivateKeyPEM  string        `mapstructure:"private_key_pem"` // base64-encoded PEM, prod fallback
	PublicKeyPEM   string        `mapstructure:"public_key_pem"`  // base64-encoded PEM, prod fallback
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
