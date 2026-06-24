package system

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"software-web-manager/backend/internal/models"
)

// SMTPConfig holds the resolved mail-sending parameters.
type SMTPConfig struct {
	SenderName     string
	SenderEmail    string
	Host           string
	Port           int
	Username       string
	Password       string
	ConnTTLSeconds int
	ForceSSL       bool
}

type smtpLoginAuth struct {
	username string
	password string
	step     int
}

func newSMTPLoginAuth(username, password string) smtp.Auth {
	return &smtpLoginAuth{username: username, password: password}
}

func (a *smtpLoginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	a.step = 0
	return "LOGIN", []byte{}, nil
}

func (a *smtpLoginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if !more {
		return nil, nil
	}
	a.step++
	if a.step == 1 {
		return []byte(a.username), nil
	}
	if a.step == 2 {
		return []byte(a.password), nil
	}
	return nil, nil
}

func smtpAuth(client *smtp.Client, cfg SMTPConfig) error {
	if strings.TrimSpace(cfg.Username) == "" {
		return nil
	}
	hasAuthMethods, methodsText := client.Extension("AUTH")
	methods := strings.ToUpper(strings.TrimSpace(methodsText))

	usePlain := !hasAuthMethods || strings.Contains(methods, "PLAIN")
	if usePlain {
		auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
		if err := client.Auth(auth); err != nil {
			return err
		}
		return nil
	}
	if strings.Contains(methods, "LOGIN") {
		if err := client.Auth(newSMTPLoginAuth(cfg.Username, cfg.Password)); err != nil {
			return err
		}
		return nil
	}
	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	return client.Auth(auth)
}

func validateSMTPPort(port int) bool {
	return port >= 1 && port <= 65535
}

func validateSMTPConnTTLSeconds(ttl int) bool {
	return ttl >= 1 && ttl <= 86400
}

func parseSenderAddress(senderName, senderEmail string) string {
	if senderEmail == "" {
		return ""
	}
	addr := mail.Address{Name: senderName, Address: senderEmail}
	return addr.String()
}

// SanitizeSMTPError trims and bounds an SMTP error before exposing it to clients.
func SanitizeSMTPError(err error) string {
	if err == nil {
		return ""
	}
	text := strings.TrimSpace(err.Error())
	if text == "" {
		return "smtp test failed"
	}
	if len(text) > 240 {
		return "smtp test failed"
	}
	return text
}

// SMTPConfigFromSettings resolves the SMTP configuration (sans password) from settings.
func (s *Service) SMTPConfigFromSettings(items map[string]models.SystemSetting) SMTPConfig {
	return SMTPConfig{
		SenderName:     GetString(items, SettingSMTPSenderNameKey, ""),
		SenderEmail:    GetString(items, SettingSMTPSenderEmailKey, ""),
		Host:           GetString(items, SettingSMTPHostKey, ""),
		Port:           GetInt(items, SettingSMTPPortKey, DefaultSMTPPort),
		Username:       GetString(items, SettingSMTPUsernameKey, ""),
		ConnTTLSeconds: GetInt(items, SettingSMTPConnTTLSecondsKey, DefaultSMTPConnTTLSeconds),
		ForceSSL:       GetBool(items, SettingSMTPForceSSLKey, DefaultSMTPForceSSL),
	}
}

// SMTPPasswordFromSettings returns the stored SMTP password and whether one is configured.
func (s *Service) SMTPPasswordFromSettings(items map[string]models.SystemSetting) (string, bool, error) {
	password := GetString(items, SettingSMTPPasswordKey, "")
	if password == "" {
		return "", false, nil
	}
	return password, true, nil
}

// ValidateSMTPConfig validates an SMTP configuration prior to sending.
func ValidateSMTPConfig(cfg SMTPConfig, passwordRequired bool) error {
	if strings.TrimSpace(cfg.Host) == "" {
		return errors.New("smtp_host required")
	}
	if !validateSMTPPort(cfg.Port) {
		return errors.New("smtp_port invalid")
	}
	if !validateSMTPConnTTLSeconds(cfg.ConnTTLSeconds) {
		return errors.New("smtp_conn_ttl_seconds invalid")
	}
	if strings.TrimSpace(cfg.SenderEmail) == "" {
		return errors.New("smtp_sender_email required")
	}
	if _, err := mail.ParseAddress(cfg.SenderEmail); err != nil {
		return errors.New("smtp_sender_email invalid")
	}
	if strings.TrimSpace(cfg.Username) == "" {
		return errors.New("smtp_username required")
	}
	if passwordRequired && strings.TrimSpace(cfg.Password) == "" {
		return errors.New("smtp_password required")
	}
	return nil
}

func sendSMTPMailOnce(cfg SMTPConfig, toEmail, subject, body string) error {
	timeout := time.Duration(cfg.ConnTTLSeconds) * time.Second
	if timeout <= 0 {
		timeout = time.Duration(DefaultSMTPConnTTLSeconds) * time.Second
	}
	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	dialer := &net.Dialer{Timeout: timeout}
	var client *smtp.Client
	var err error
	if cfg.ForceSSL {
		tlsConn, tlsErr := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{ServerName: cfg.Host, MinVersion: tls.VersionTLS12})
		if tlsErr != nil {
			return fmt.Errorf("dial smtp server failed")
		}
		client, err = smtp.NewClient(tlsConn, cfg.Host)
		if err != nil {
			return fmt.Errorf("create smtp ssl client failed")
		}
	} else {
		conn, dialErr := dialer.Dial("tcp", addr)
		if dialErr != nil {
			return fmt.Errorf("dial smtp server failed")
		}
		client, err = smtp.NewClient(conn, cfg.Host)
		if err != nil {
			_ = conn.Close()
			return fmt.Errorf("create smtp client failed")
		}
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{ServerName: cfg.Host, MinVersion: tls.VersionTLS12}); err != nil {
				return fmt.Errorf("smtp starttls failed")
			}
		}
	}
	defer func() {
		_ = client.Quit()
		_ = client.Close()
	}()

	if err := smtpAuth(client, cfg); err != nil {
		return fmt.Errorf("smtp auth failed: %v", err)
	}
	if err := client.Mail(cfg.SenderEmail); err != nil {
		return fmt.Errorf("smtp set sender failed")
	}
	if err := client.Rcpt(toEmail); err != nil {
		return fmt.Errorf("smtp set recipient failed")
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data command failed")
	}
	from := parseSenderAddress(cfg.SenderName, cfg.SenderEmail)
	if from == "" {
		from = cfg.SenderEmail
	}
	content := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s", from, toEmail, subject, body)
	if _, err := writer.Write([]byte(content)); err != nil {
		_ = writer.Close()
		return fmt.Errorf("smtp write message failed")
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("smtp finalize message failed")
	}
	return nil
}

func shouldRetrySMTPError(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(strings.TrimSpace(err.Error()))
	if text == "" {
		return false
	}
	if strings.Contains(text, "system busy") {
		return true
	}
	if strings.Contains(text, "too many") {
		return true
	}
	return false
}

// SendMail sends a plain-text message, retrying transient failures.
func SendMail(cfg SMTPConfig, toEmail, subject, body string) error {
	var lastErr error
	for i := 0; i < 3; i++ {
		lastErr = sendSMTPMailOnce(cfg, toEmail, subject, body)
		if lastErr == nil {
			return nil
		}
		if !shouldRetrySMTPError(lastErr) || i == 2 {
			return lastErr
		}
		time.Sleep(time.Duration(i+1) * time.Second)
	}
	return lastErr
}

// TestSMTPRequest is the payload accepted by the SMTP test endpoint.
type TestSMTPRequest struct {
	ToEmail            string  `json:"to_email" binding:"required,email"`
	Subject            *string `json:"subject"`
	Body               *string `json:"body"`
	SMTPSenderName     *string `json:"smtp_sender_name"`
	SMTPSenderEmail    *string `json:"smtp_sender_email"`
	SMTPHost           *string `json:"smtp_host"`
	SMTPPort           *int    `json:"smtp_port"`
	SMTPUsername       *string `json:"smtp_username"`
	SMTPPassword       *string `json:"smtp_password"`
	SMTPConnTTLSeconds *int    `json:"smtp_conn_ttl_seconds"`
	SMTPForceSSL       *bool   `json:"smtp_force_ssl"`
}

// SendTestMail validates the effective SMTP configuration and sends a test message.
// Validation/send failures are returned as *ValidationError (client errors); table and
// load failures are returned as plain errors (server errors).
func (s *Service) SendTestMail(req TestSMTPRequest) error {
	if !s.HasSettingsTable() {
		if err := s.DB.AutoMigrate(&models.SystemSetting{}); err != nil {
			return errors.New("failed to initialize system settings table")
		}
	}
	items, err := s.ListSettings()
	if err != nil {
		return errors.New("failed to load system settings")
	}
	cfg := s.SMTPConfigFromSettings(items)
	password, configured, passwordErr := s.SMTPPasswordFromSettings(items)
	if passwordErr != nil {
		return &ValidationError{Message: "failed to load smtp password"}
	}
	cfg.Password = password

	if req.SMTPSenderName != nil {
		cfg.SenderName = strings.TrimSpace(*req.SMTPSenderName)
	}
	if req.SMTPSenderEmail != nil {
		cfg.SenderEmail = strings.TrimSpace(*req.SMTPSenderEmail)
	}
	if req.SMTPHost != nil {
		cfg.Host = strings.TrimSpace(*req.SMTPHost)
	}
	if req.SMTPPort != nil {
		cfg.Port = *req.SMTPPort
	}
	if req.SMTPUsername != nil {
		cfg.Username = strings.TrimSpace(*req.SMTPUsername)
	}
	if req.SMTPConnTTLSeconds != nil {
		cfg.ConnTTLSeconds = *req.SMTPConnTTLSeconds
	}
	if req.SMTPForceSSL != nil {
		cfg.ForceSSL = *req.SMTPForceSSL
	}
	if req.SMTPPassword != nil && strings.TrimSpace(*req.SMTPPassword) != "" {
		cfg.Password = strings.TrimSpace(*req.SMTPPassword)
		configured = true
	}
	if err := ValidateSMTPConfig(cfg, !configured); err != nil {
		return &ValidationError{Message: err.Error()}
	}

	subject := "SMTP 测试邮件"
	if req.Subject != nil && strings.TrimSpace(*req.Subject) != "" {
		subject = strings.TrimSpace(*req.Subject)
	}
	body := "这是一封来自系统设置的测试邮件。"
	if req.Body != nil && strings.TrimSpace(*req.Body) != "" {
		body = strings.TrimSpace(*req.Body)
	}
	if err := SendMail(cfg, strings.TrimSpace(req.ToEmail), subject, body); err != nil {
		return &ValidationError{Message: SanitizeSMTPError(err)}
	}
	return nil
}
