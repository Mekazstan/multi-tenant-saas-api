package email

import (
	"bytes"
	"fmt"
	"html/template"
	"net/smtp"
	"os"
	"path/filepath"
)

type EmailService struct {
	smtpHost     string
	smtpPort     string
	smtpUsername string
	smtpPassword string
	fromEmail    string
	fromName     string
	templates    map[string]*template.Template
}

type EmailData struct {
	To          string
	Subject     string
	TemplateKey string
	Data        interface{}
}

func NewEmailService() (*EmailService, error) {
	service := &EmailService{
		smtpHost:     os.Getenv("SMTP_HOST"),
		smtpPort:     os.Getenv("SMTP_PORT"),
		smtpUsername: os.Getenv("SMTP_USERNAME"),
		smtpPassword: os.Getenv("SMTP_PASSWORD"),
		fromEmail:    os.Getenv("FROM_EMAIL"),
		fromName:     os.Getenv("FROM_NAME"),
		templates:    make(map[string]*template.Template),
	}

	if err := service.loadTemplates(); err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	return service, nil
}

func (s *EmailService) loadTemplates() error {
	templateDir := "internal/email/templates"

	templates := map[string]string{
		"welcome":            "welcome.html",
		"email_verification": "email_verification.html",
		"password_reset":     "password_reset.html",
		"billing_invoice":    "billing_invoice.html",
		"payment_success":    "payment_success.html",
		"team_invitation":    "team_invitation.html",
		"overdue_payment":    "overdue_payment.html",
	}

	for key, filename := range templates {
		path := filepath.Join(templateDir, filename)
		tmpl, err := template.ParseFiles(path)
		if err != nil {
			tmpl = template.Must(template.New(key).Parse(defaultTemplate))
		}
		s.templates[key] = tmpl
	}

	return nil
}

func (s *EmailService) SendEmail(data EmailData) error {
	tmpl, ok := s.templates[data.TemplateKey]
	if !ok {
		return fmt.Errorf("template %s not found", data.TemplateKey)
	}

	var body bytes.Buffer
	if err := tmpl.Execute(&body, data.Data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	message := fmt.Sprintf("From: %s <%s>\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: text/html; charset=UTF-8\r\n"+
		"\r\n"+
		"%s", s.fromName, s.fromEmail, data.To, data.Subject, body.String())

	auth := smtp.PlainAuth("", s.smtpUsername, s.smtpPassword, s.smtpHost)
	addr := fmt.Sprintf("%s:%s", s.smtpHost, s.smtpPort)

	err := smtp.SendMail(addr, auth, s.fromEmail, []string{data.To}, []byte(message))
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

type WelcomeEmailData struct {
	Name             string
	OrganizationName string
	DashboardURL     string
}

func (s *EmailService) SendWelcomeEmail(to, name, orgName string) error {
	return s.SendEmail(EmailData{
		To:          to,
		Subject:     fmt.Sprintf("Welcome to %s! 🎉", orgName),
		TemplateKey: "welcome",
		Data: WelcomeEmailData{
			Name:             name,
			OrganizationName: orgName,
			DashboardURL:     os.Getenv("APP_URL") + "/dashboard",
		},
	})
}

type EmailVerificationData struct {
	Name            string
	VerificationURL string
	ExpiresIn       string
}

func (s *EmailService) SendEmailVerification(to, name, token string) error {
	verificationURL := fmt.Sprintf("%s/verify-email?token=%s", os.Getenv("APP_URL"), token)

	return s.SendEmail(EmailData{
		To:          to,
		Subject:     "Verify your email address",
		TemplateKey: "email_verification",
		Data: EmailVerificationData{
			Name:            name,
			VerificationURL: verificationURL,
			ExpiresIn:       "24 hours",
		},
	})
}

type PasswordResetData struct {
	Name      string
	ResetURL  string
	ExpiresIn string
}

func (s *EmailService) SendPasswordReset(to, name, token string) error {
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", os.Getenv("APP_URL"), token)

	return s.SendEmail(EmailData{
		To:          to,
		Subject:     "Reset your password",
		TemplateKey: "password_reset",
		Data: PasswordResetData{
			Name:      name,
			ResetURL:  resetURL,
			ExpiresIn: "1 hour",
		},
	})
}

type BillingInvoiceData struct {
	OrganizationName string
	InvoiceNumber    string
	PeriodStart      string
	PeriodEnd        string
	TotalRequests    int
	TotalAmount      float64
	InvoiceURL       string
	DueDate          string
}

func (s *EmailService) SendBillingInvoice(to string, data BillingInvoiceData) error {
	return s.SendEmail(EmailData{
		To:          to,
		Subject:     fmt.Sprintf("Invoice #%s - Your monthly usage", data.InvoiceNumber),
		TemplateKey: "billing_invoice",
		Data:        data,
	})
}

type PaymentSuccessData struct {
	OrganizationName string
	Amount           float64
	InvoiceNumber    string
	ReceiptURL       string
}

func (s *EmailService) SendPaymentSuccess(to string, data PaymentSuccessData) error {
	return s.SendEmail(EmailData{
		To:          to,
		Subject:     "Payment received - Thank you!",
		TemplateKey: "payment_success",
		Data:        data,
	})
}

type TeamInvitationData struct {
	InviterName      string
	OrganizationName string
	Role             string
	InvitationURL    string
	ExpiresIn        string
}

func (s *EmailService) SendTeamInvitation(to string, data TeamInvitationData) error {
	return s.SendEmail(EmailData{
		To:          to,
		Subject:     fmt.Sprintf("You've been invited to join %s", data.OrganizationName),
		TemplateKey: "team_invitation",
		Data:        data,
	})
}

type OverduePaymentData struct {
	OrganizationName string
	InvoiceNumber    string
	Amount           float64
	DaysPastDue      int
	PaymentURL       string
}

func (s *EmailService) SendOverduePayment(to string, data OverduePaymentData) error {
	return s.SendEmail(EmailData{
		To:          to,
		Subject:     fmt.Sprintf("Overdue Payment Notice - Invoice #%s", data.InvoiceNumber),
		TemplateKey: "overdue_payment",
		Data:        data,
	})
}

const defaultTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Email</title>
</head>
<body>
    <p>{{.}}</p>
</body>
</html>
`
