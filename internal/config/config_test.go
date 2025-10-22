package config

import (
	"os"
	"testing"
)

func TestConfigLoad(t *testing.T) {
	os.Setenv("DATABASE_URL", "postgres://test")
	os.Setenv("JWT_SECRET", "test-secret")
	defer os.Unsetenv("DATABASE_URL")
	defer os.Unsetenv("JWT_SECRET")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.DatabaseURL != "postgres://test" {
		t.Errorf("Expected DatabaseURL 'postgres://test', got '%s'", cfg.DatabaseURL)
	}

	if cfg.JWTSecret != "test-secret" {
		t.Errorf("Expected JWTSecret 'test-secret', got '%s'", cfg.JWTSecret)
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "Valid config",
			config: &Config{
				DatabaseURL: "postgres://test",
				JWTSecret:   "secret",
			},
			wantErr: false,
		},
		{
			name: "Missing database URL",
			config: &Config{
				JWTSecret: "secret",
			},
			wantErr: true,
		},
		{
			name: "Missing JWT secret",
			config: &Config{
				DatabaseURL: "postgres://test",
			},
			wantErr: true,
		},
		{
			name: "Incomplete SMTP config",
			config: &Config{
				DatabaseURL: "postgres://test",
				JWTSecret:   "secret",
				SMTPHost:    "smtp.gmail.com",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEnvironmentHelpers(t *testing.T) {
	cfg := &Config{Environment: Development}
	if !cfg.IsDevelopment() {
		t.Error("Expected IsDevelopment() to be true")
	}
	if cfg.IsProduction() {
		t.Error("Expected IsProduction() to be false")
	}

	cfg.Environment = Production
	if cfg.IsDevelopment() {
		t.Error("Expected IsDevelopment() to be false")
	}
	if !cfg.IsProduction() {
		t.Error("Expected IsProduction() to be true")
	}
}
