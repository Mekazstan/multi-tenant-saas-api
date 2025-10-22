package email

import (
	"os"
	"testing"
)

func TestEmailServiceCreation(t *testing.T) {
	os.Setenv("SMTP_HOST", "smtp.gmail.com")
	os.Setenv("SMTP_PORT", "587")
	os.Setenv("SMTP_USERNAME", "test@example.com")
	os.Setenv("SMTP_PASSWORD", "password")
	os.Setenv("FROM_EMAIL", "noreply@example.com")
	os.Setenv("FROM_NAME", "Test Service")
	
	defer func() {
		os.Unsetenv("SMTP_HOST")
		os.Unsetenv("SMTP_PORT")
		os.Unsetenv("SMTP_USERNAME")
		os.Unsetenv("SMTP_PASSWORD")
		os.Unsetenv("FROM_EMAIL")
		os.Unsetenv("FROM_NAME")
	}()

	service, err := NewEmailService()
	if err != nil {
		t.Fatalf("NewEmailService() error = %v", err)
	}

	if service == nil {
		t.Fatal("NewEmailService() returned nil")
	}

	if service.smtpHost != "smtp.gmail.com" {
		t.Errorf("Expected smtpHost 'smtp.gmail.com', got '%s'", service.smtpHost)
	}
}
