// Package config loads and validates WGA deployment configuration.
package config

import (
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"
)

// Lookup retrieves a configuration value by name.
type Lookup func(string) string

type parsed[T any] struct {
	value T
	err   error
}

// Secret holds a sensitive configuration value and redacts it when formatted.
type Secret struct {
	value string
}

// Value returns the unredacted secret value.
func (s Secret) Value() string {
	return s.value
}

// String returns a redacted representation of the secret.
func (Secret) String() string {
	return "[redacted]"
}

// GoString returns a redacted Go-syntax representation of the secret.
func (Secret) GoString() string {
	return "config.Secret([redacted])"
}

// Environment identifies the deployment environment.
type Environment string

// Supported deployment environments.
const (
	EnvironmentDevelopment Environment = "development"
	EnvironmentTest        Environment = "test"
	EnvironmentStaging     Environment = "staging"
	EnvironmentProduction  Environment = "production"
)

// IsDevelopment reports whether the environment is for local development.
func (e Environment) IsDevelopment() bool {
	return e == EnvironmentDevelopment
}

// AllowsCaptchaBypass reports whether CAPTCHA verification may be disabled.
func (e Environment) AllowsCaptchaBypass() bool {
	return e == EnvironmentDevelopment || e == EnvironmentTest
}

// PublicURL is the canonical external URL for the application.
type PublicURL struct {
	url url.URL
}

// String returns the canonical public URL.
func (u PublicURL) String() string {
	return u.url.String()
}

// Resolve returns path resolved against the public URL.
func (u PublicURL) Resolve(path string) string {
	if u.url.Scheme == "" || u.url.Host == "" {
		return path
	}

	reference, err := url.Parse(path)
	if err != nil {
		return u.String()
	}

	reference.Scheme = ""
	reference.Opaque = ""
	reference.User = nil
	reference.Host = ""

	if reference.Path == "" {
		reference.Path = "/"
	} else if !strings.HasPrefix(reference.Path, "/") {
		reference.Path = "/" + reference.Path
	}

	return u.url.ResolveReference(reference).String()
}

// Storage defines the optional S3-compatible object storage connection.
type Storage struct {
	Enabled        bool
	Endpoint       url.URL
	Bucket         string
	Region         string
	AccessKey      string
	AccessSecret   Secret
	ForcePathStyle bool
}

// MailSender identifies the sender of application email.
type MailSender struct {
	Name    string
	Address mail.Address
}

// SMTP defines the SMTP connection used to send application email.
type SMTP struct {
	Host     string
	Port     int
	Username string
	Password Secret
}

// Mail contains the application email configuration.
type Mail struct {
	Sender MailSender
	SMTP   SMTP
}

// Administrator defines an optional administrator created during migration.
type Administrator struct {
	Email    mail.Address
	Password Secret
	Enabled  bool
}

// Captcha contains the reCAPTCHA settings used by protected routes.
type Captcha struct {
	secret  Secret
	siteKey string
	verify  bool
}

// Sentry contains the optional Sentry monitoring configuration.
type Sentry struct {
	dsn Secret
}

// DSN returns the configured Sentry DSN.
func (s Sentry) DSN() string {
	return s.dsn.Value()
}

// String returns a redacted representation of the Sentry configuration.
func (Sentry) String() string {
	return "[redacted]"
}

// GoString returns a redacted Go-syntax representation of the Sentry configuration.
func (Sentry) GoString() string {
	return "config.Sentry([redacted])"
}

// Verify reports whether CAPTCHA verification is enabled.
func (c Captcha) Verify() bool {
	return c.verify
}

// Secret returns the configured reCAPTCHA secret.
func (c Captcha) Secret() string {
	return c.secret.Value()
}

// SiteKey returns the configured reCAPTCHA site key.
func (c Captcha) SiteKey() string {
	return c.siteKey
}

// Postcards contains the postcard delivery schedule and email settings.
type Postcards struct {
	expression string
	schedule   cron.Schedule
	Sender     MailSender
	PublicURL  PublicURL
}

// Expression returns the configured postcard delivery schedule.
func (p Postcards) Expression() string {
	return p.expression
}

// Sitemap contains the settings required to generate a sitemap.
type Sitemap struct {
	Environment Environment
	PublicURL   PublicURL
}

// Server contains the settings required to run the HTTP application.
type Server struct {
	Environment Environment
	PublicURL   PublicURL
	Postcards   Postcards
	Captcha     Captcha
	Sentry      Sentry
}

// Sitemap returns the sitemap settings derived from the server settings.
func (s Server) Sitemap() Sitemap {
	return Sitemap{
		Environment: s.Environment,
		PublicURL:   s.PublicURL,
	}
}

// InitialSettings contains configuration applied by the initial migration.
type InitialSettings struct {
	PublicURL PublicURL
	Storage   Storage
	Mail      Mail
}

// Migrations provides configuration needed while applying migrations.
type Migrations struct {
	publicURL     parsed[PublicURL]
	storage       parsed[Storage]
	mail          parsed[Mail]
	administrator parsed[Administrator]
}

// InitialSettings returns validated settings for the initial migration.
func (m Migrations) InitialSettings() (InitialSettings, error) {
	storage := m.storage.value
	storage.Enabled = m.storage.err == nil

	settings := InitialSettings{
		PublicURL: m.publicURL.value,
		Storage:   storage,
		Mail:      m.mail.value,
	}

	return settings, errors.Join(
		m.publicURL.err,
		m.mail.err,
		requireMigrationMail(settings.Mail),
	)
}

// Administrator returns the optional administrator bootstrap configuration.
func (m Migrations) Administrator() (Administrator, error) {
	return m.administrator.value, m.administrator.err
}

// Config holds parsed application configuration for each runtime capability.
type Config struct {
	environment parsed[Environment]
	publicURL   parsed[PublicURL]
	sender      parsed[MailSender]
	postcards   parsed[Postcards]
	captcha     Captcha
	sentry      Sentry
	migrations  Migrations
}

// Load reads .env when present, then loads configuration from the environment.
func Load() (Config, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, errors.New("unable to load .env")
	}

	return LoadFrom(os.Getenv), nil
}

// LoadFrom loads configuration values from lookup without reading the process environment.
func LoadFrom(lookup Lookup) Config {
	if lookup == nil {
		lookup = func(string) string { return "" }
	}

	environment := parseEnvironment(lookup("WGA_ENV"))
	publicURL := parsePublicURL(lookup)
	sender := parseSender(lookup)
	mailConfig := parseMail(lookup, sender)
	storage := parseStorage(lookup)
	administrator := parseAdministrator(lookup)
	postcards := parsePostcards(lookup, publicURL.value, sender.value)
	captcha := Captcha{
		secret:  Secret{value: lookup("WGA_RECAPTCHA_SECRET")},
		siteKey: lookup("WGA_RECAPTCHA_SITE_KEY"),
	}
	captcha.verify = captcha.secret.Value() != ""

	return Config{
		environment: environment,
		publicURL:   publicURL,
		sender:      sender,
		postcards:   postcards,
		captcha:     captcha,
		sentry:      Sentry{dsn: Secret{value: lookup("WGA_SENTRY_DSN")}},
		migrations: Migrations{
			publicURL:     publicURL,
			storage:       storage,
			mail:          mailConfig,
			administrator: administrator,
		},
	}
}

// Environment returns the configured deployment environment.
func (c Config) Environment() Environment {
	return c.environment.value
}

// Server returns validated settings required to run the HTTP application.
func (c Config) Server() (Server, error) {
	server := Server{
		Environment: c.environment.value,
		PublicURL:   c.publicURL.value,
		Postcards:   c.postcards.value,
		Captcha:     c.captcha,
		Sentry:      c.sentry,
	}

	senderErr := c.sender.err
	if senderErr == nil {
		if c.sender.value.Name == "" {
			senderErr = errors.Join(senderErr, required("WGA_SENDER_NAME"))
		}
		if c.sender.value.Address.Address == "" {
			senderErr = errors.Join(senderErr, required("WGA_SENDER_ADDRESS"))
		}
	}

	var captchaErr error
	if c.environment.err == nil && !c.environment.value.AllowsCaptchaBypass() {
		if !c.captcha.Verify() {
			captchaErr = errors.Join(captchaErr, required("WGA_RECAPTCHA_SECRET"))
		}
		if c.captcha.SiteKey() == "" {
			captchaErr = errors.Join(captchaErr, required("WGA_RECAPTCHA_SITE_KEY"))
		}
	}

	return server, errors.Join(
		c.environment.err,
		c.publicURL.err,
		c.postcards.err,
		senderErr,
		captchaErr,
	)
}

// Sitemap returns validated settings required to generate a sitemap.
func (c Config) Sitemap() (Sitemap, error) {
	return Sitemap{
		Environment: c.environment.value,
		PublicURL:   c.publicURL.value,
	}, errors.Join(c.environment.err, c.publicURL.err)
}

// Migrations returns the configuration needed while applying migrations.
func (c Config) Migrations() Migrations {
	return c.migrations
}

// parseEnvironment validates a deployment environment value.
func parseEnvironment(value string) parsed[Environment] {
	switch Environment(value) {
	case EnvironmentDevelopment, EnvironmentTest, EnvironmentStaging, EnvironmentProduction:
		return parsed[Environment]{value: Environment(value)}
	default:
		return parsed[Environment]{err: fmt.Errorf("WGA_ENV must be one of development, test, staging, production")}
	}
}

// parsePublicURL validates the configured public protocol and hostname.
func parsePublicURL(lookup Lookup) parsed[PublicURL] {
	protocol := lookup("WGA_PROTOCOL")
	if protocol == "" {
		return parsed[PublicURL]{err: required("WGA_PROTOCOL")}
	}
	if protocol != "http" && protocol != "https" {
		return parsed[PublicURL]{err: fmt.Errorf("WGA_PROTOCOL must be http or https")}
	}

	hostname := lookup("WGA_HOSTNAME")
	if hostname == "" {
		return parsed[PublicURL]{err: required("WGA_HOSTNAME")}
	}

	parsedURL, err := url.Parse(protocol + "://" + hostname)
	if err != nil || parsedURL.Host == "" || parsedURL.Hostname() == "" || parsedURL.User != nil || parsedURL.Path != "" || parsedURL.RawQuery != "" || parsedURL.Fragment != "" {
		return parsed[PublicURL]{err: fmt.Errorf("WGA_HOSTNAME must be a host with an optional port")}
	}

	return parsed[PublicURL]{value: PublicURL{url: *parsedURL}}
}

// parseStorage reads the optional S3-compatible storage settings.
func parseStorage(lookup Lookup) parsed[Storage] {
	endpoint, endpointErr := parseAbsoluteURL("WGA_S3_ENDPOINT", lookup("WGA_S3_ENDPOINT"))

	storage := Storage{
		Endpoint:       endpoint,
		Bucket:         lookup("WGA_S3_BUCKET"),
		Region:         lookup("WGA_S3_REGION"),
		AccessKey:      lookup("WGA_S3_ACCESS_KEY"),
		AccessSecret:   Secret{value: lookup("WGA_S3_ACCESS_SECRET")},
		ForcePathStyle: true,
	}

	return parsed[Storage]{
		value: storage,
		err: errors.Join(
			endpointErr,
			requireValue("WGA_S3_BUCKET", storage.Bucket),
			requireValue("WGA_S3_ACCESS_KEY", storage.AccessKey),
			requireValue("WGA_S3_ACCESS_SECRET", storage.AccessSecret.Value()),
		),
	}
}

// parseMail reads SMTP settings and combines them with sender configuration.
func parseMail(lookup Lookup, sender parsed[MailSender]) parsed[Mail] {

	smtp := SMTP{
		Host:     lookup("WGA_SMTP_HOST"),
		Username: lookup("WGA_SMTP_USERNAME"),
		Password: Secret{value: lookup("WGA_SMTP_PASSWORD")},
	}

	portValue := lookup("WGA_SMTP_PORT")
	var portErr error
	if portValue != "" {
		port, err := strconv.ParseUint(portValue, 10, 16)
		if err != nil || port == 0 {
			portErr = fmt.Errorf("WGA_SMTP_PORT must be an integer between 1 and 65535")
		} else {
			smtp.Port = int(port)
		}
	}

	return parsed[Mail]{
		value: Mail{
			Sender: sender.value,
			SMTP:   smtp,
		},
		err: errors.Join(sender.err, portErr),
	}
}

// parseSender validates the configured email sender.
func parseSender(lookup Lookup) parsed[MailSender] {
	sender := MailSender{Name: lookup("WGA_SENDER_NAME")}
	address := lookup("WGA_SENDER_ADDRESS")
	if address == "" {
		return parsed[MailSender]{value: sender}
	}

	parsedAddress, err := mail.ParseAddress(address)
	if err != nil || parsedAddress.Address == "" {
		return parsed[MailSender]{
			value: sender,
			err:   fmt.Errorf("WGA_SENDER_ADDRESS must be a valid email address"),
		}
	}

	sender.Address = *parsedAddress
	return parsed[MailSender]{value: sender}
}

// parseAdministrator validates the optional administrator bootstrap credentials.
func parseAdministrator(lookup Lookup) parsed[Administrator] {
	email := lookup("WGA_ADMIN_EMAIL")
	password := lookup("WGA_ADMIN_PASSWORD")
	if email == "" && password == "" {
		return parsed[Administrator]{}
	}
	if email == "" || password == "" {
		return parsed[Administrator]{err: fmt.Errorf("WGA_ADMIN_EMAIL and WGA_ADMIN_PASSWORD must be set together")}
	}

	parsedEmail, err := mail.ParseAddress(email)
	if err != nil || parsedEmail.Address == "" {
		return parsed[Administrator]{err: fmt.Errorf("WGA_ADMIN_EMAIL must be a valid email address")}
	}

	return parsed[Administrator]{
		value: Administrator{
			Email:    *parsedEmail,
			Password: Secret{value: password},
			Enabled:  true,
		},
	}
}

// parsePostcards validates the postcard delivery schedule and dependencies.
func parsePostcards(lookup Lookup, publicURL PublicURL, sender MailSender) parsed[Postcards] {
	expression := lookup("WGA_POSTCARD_FREQUENCY")
	if expression == "" {
		expression = "*/1 * * * *"
	}

	schedule, err := cron.ParseStandard(expression)
	if err != nil {
		return parsed[Postcards]{err: fmt.Errorf("WGA_POSTCARD_FREQUENCY must be a valid cron expression")}
	}

	return parsed[Postcards]{
		value: Postcards{
			expression: expression,
			schedule:   schedule,
			Sender:     sender,
			PublicURL:  publicURL,
		},
	}
}

// parseAbsoluteURL validates value as an absolute URL for name.
func parseAbsoluteURL(name string, value string) (url.URL, error) {
	if value == "" {
		return url.URL{}, required(name)
	}

	parsedURL, err := url.Parse(value)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return url.URL{}, fmt.Errorf("%s must be an absolute URL", name)
	}

	return *parsedURL, nil
}

// required returns an error indicating that name is not configured.
func required(name string) error {
	return fmt.Errorf("%s must be set", name)
}

// requireValue returns an error when value is empty.
func requireValue(name string, value string) error {
	if value == "" {
		return required(name)
	}

	return nil
}

// requireMigrationMail validates the mail settings required by migrations.
func requireMigrationMail(mail Mail) error {
	var errs []error
	if mail.Sender.Name == "" {
		errs = append(errs, required("WGA_SENDER_NAME"))
	}
	if mail.Sender.Address.Address == "" {
		errs = append(errs, required("WGA_SENDER_ADDRESS"))
	}
	if mail.SMTP.Host == "" {
		errs = append(errs, required("WGA_SMTP_HOST"))
	}
	if mail.SMTP.Port == 0 {
		errs = append(errs, required("WGA_SMTP_PORT"))
	}

	return errors.Join(errs...)
}
