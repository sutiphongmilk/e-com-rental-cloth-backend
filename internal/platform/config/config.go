package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	AppEnv             string   `env:"APP_ENV" envDefault:"development"`
	HTTPAddr           string   `env:"HTTP_ADDR" envDefault:"127.0.0.1:8080"`
	DatabaseURL        string   `env:"DATABASE_URL,required,notEmpty"`
	CORSAllowedOrigins []string `env:"CORS_ALLOWED_ORIGINS" envSeparator:"," envDefault:"http://localhost:3000"`
	SessionCookieName  string   `env:"SESSION_COOKIE_NAME" envDefault:"rental_sid"`
	SessionTTLHours    int      `env:"SESSION_TTL_HOURS" envDefault:"168"`
	CookieSecure       bool     `env:"COOKIE_SECURE"`
	TurnstileSecretKey string   `env:"TURNSTILE_SECRET_KEY"`
}

func Load() (Config, error) {
	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return Config{}, fmt.Errorf("parse environment config: %w", err)
	}
	return cfg, nil
}

func (c Config) IsProduction() bool {
	return c.AppEnv == "production"
}

func (c Config) UseSecureCookie() bool {
	return c.IsProduction() || c.CookieSecure
}

func (c Config) SessionTTL() time.Duration {
	return time.Duration(c.SessionTTLHours) * time.Hour
}
