// Package config loads and validates WGA deployment configuration.
package config

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/getsentry/sentry-go"
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

// ClientIPSource selects how the trusted client identity is resolved for
// anonymous-write surfaces. direct parses the socket peer and ignores
// forwarding headers; railway follows the production Railway-edge contract.
type ClientIPSource string

// Supported client-identity sources.
const (
	ClientIPSourceDirect  ClientIPSource = "direct"
	ClientIPSourceRailway ClientIPSource = "railway"
)

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
	dsn        Secret
	browserDSN Secret
}

// DSN returns the configured server Sentry DSN.
func (s Sentry) DSN() string {
	return s.dsn.Value()
}

// BrowserDSN returns the configured browser Sentry DSN.
func (s Sentry) BrowserDSN() string {
	return s.browserDSN.Value()
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
	expression   string
	schedule     cron.Schedule
	tokenKeyring PostcardTokenKeyring
	Sender       MailSender
	PublicURL    PublicURL
}

// Expression returns the configured postcard delivery schedule.
func (p Postcards) Expression() string {
	return p.expression
}

// TokenKeyring returns the keyring used to protect postcard access tokens.
func (p Postcards) TokenKeyring() PostcardTokenKeyring {
	return p.tokenKeyring
}

// PostcardTokenKey holds an AES-256 key and redacts it when formatted.
type PostcardTokenKey struct {
	value [32]byte
}

// Bytes returns a copy of the key bytes.
func (k PostcardTokenKey) Bytes() []byte {
	value := make([]byte, len(k.value))
	copy(value, k.value[:])
	return value
}

// String returns a redacted representation of the key.
func (PostcardTokenKey) String() string {
	return "[redacted]"
}

// GoString returns a redacted Go-syntax representation of the key.
func (PostcardTokenKey) GoString() string {
	return "config.PostcardTokenKey([redacted])"
}

// PostcardTokenKeyring contains versioned postcard token keys.
type PostcardTokenKeyring struct {
	activeKeyID string
	keys        map[string]PostcardTokenKey
}

// ActiveKeyID returns the identifier used when issuing new tokens.
func (k PostcardTokenKeyring) ActiveKeyID() string {
	return k.activeKeyID
}

// ActiveKey returns the key used when issuing new tokens.
func (k PostcardTokenKeyring) ActiveKey() PostcardTokenKey {
	return k.keys[k.activeKeyID]
}

// Key returns the key identified by keyID for token recovery.
func (k PostcardTokenKeyring) Key(keyID string) (PostcardTokenKey, bool) {
	key, ok := k.keys[keyID]
	return key, ok
}

// String returns a redacted representation of the keyring.
func (PostcardTokenKeyring) String() string {
	return "[redacted]"
}

// GoString returns a redacted Go-syntax representation of the keyring.
func (PostcardTokenKeyring) GoString() string {
	return "config.PostcardTokenKeyring([redacted])"
}

// Sitemap contains the settings required to generate a sitemap.
type Sitemap struct {
	Environment Environment
	PublicURL   PublicURL
}

// Server contains the settings required to run the HTTP application.
type Server struct {
	Environment    Environment
	PublicURL      PublicURL
	ClientIPSource ClientIPSource
	Postcards      Postcards
	Captcha        Captcha
	Sentry         Sentry
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
	publicURL      parsed[PublicURL]
	storage        parsed[Storage]
	mail           parsed[Mail]
	administrator  parsed[Administrator]
	seedSQLitePath string
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

// SeedSQLitePath returns the optional authoritative source database for a fresh bootstrap.
func (m Migrations) SeedSQLitePath() string {
	return m.seedSQLitePath
}

// Config holds parsed application configuration for each runtime capability.
type Config struct {
	environment          parsed[Environment]
	publicURL            parsed[PublicURL]
	clientIPSource       parsed[ClientIPSource]
	sender               parsed[MailSender]
	postcards            parsed[Postcards]
	postcardTokenKeyring parsed[PostcardTokenKeyring]
	captcha              Captcha
	sentry               parsed[Sentry]
	migrations           Migrations
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
	clientIPSource := parseClientIPSource(lookup, environment.value)
	sender := parseSender(lookup)
	mailConfig := parseMail(lookup, sender)
	storage := parseStorage(lookup)
	administrator := parseAdministrator(lookup)
	postcardTokenKeyring := parsePostcardTokenKeyring(
		Secret{value: lookup("WGA_POSTCARD_TOKEN_KEYS")},
		lookup("WGA_POSTCARD_TOKEN_ACTIVE_KEY_ID"),
	)
	postcards := parsePostcards(lookup, publicURL.value, sender.value, postcardTokenKeyring.value)
	sentryConfig := parseSentry(
		lookup("WGA_SENTRY_DSN"),
		lookup("WGA_SENTRY_BROWSER_DSN"),
	)
	captcha := Captcha{
		secret:  Secret{value: lookup("WGA_RECAPTCHA_SECRET")},
		siteKey: lookup("WGA_RECAPTCHA_SITE_KEY"),
	}
	captcha.verify = captcha.secret.Value() != ""

	return Config{
		environment:          environment,
		publicURL:            publicURL,
		clientIPSource:       clientIPSource,
		sender:               sender,
		postcards:            postcards,
		postcardTokenKeyring: postcardTokenKeyring,
		captcha:              captcha,
		sentry:               sentryConfig,
		migrations: Migrations{
			publicURL:      publicURL,
			storage:        storage,
			mail:           mailConfig,
			administrator:  administrator,
			seedSQLitePath: lookup("WGA_SEED_SQLITE_PATH"),
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
		Environment:    c.environment.value,
		PublicURL:      c.publicURL.value,
		ClientIPSource: c.clientIPSource.value,
		Postcards:      c.postcards.value,
		Captcha:        c.captcha,
		Sentry:         c.sentry.value,
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
		c.clientIPSource.err,
		c.postcards.err,
		c.sentry.err,
		senderErr,
		captchaErr,
	)
}

// PostcardTokenKeyring returns the validated keyring used for postcard access tokens.
func (c Config) PostcardTokenKeyring() (PostcardTokenKeyring, error) {
	return c.postcardTokenKeyring.value, c.postcardTokenKeyring.err
}

// parseSentry validates the optional server and browser Sentry DSNs.
func parseSentry(dsn string, browserDSN string) parsed[Sentry] {
	settings := Sentry{
		dsn:        Secret{value: dsn},
		browserDSN: Secret{value: browserDSN},
	}

	return parsed[Sentry]{
		value: settings,
		err: errors.Join(
			validateSentryDSN("WGA_SENTRY_DSN", dsn),
			validateSentryDSN("WGA_SENTRY_BROWSER_DSN", browserDSN),
		),
	}
}

func validateSentryDSN(name string, value string) error {
	if value == "" {
		return nil
	}

	parsedURL, err := url.Parse(value)
	if err == nil {
		if _, hasSecret := parsedURL.User.Password(); hasSecret {
			return fmt.Errorf("%s must not contain a secret key", name)
		}
	}

	if _, err := sentry.NewDsn(value); err != nil {
		return fmt.Errorf("%s must be a valid Sentry DSN", name)
	}

	return nil
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

// parseClientIPSource validates the trusted client-identity source. An empty
// value defaults to direct for development and test, and is otherwise an error
// so production and staging must select a source explicitly.
func parseClientIPSource(lookup Lookup, environment Environment) parsed[ClientIPSource] {
	value := lookup("WGA_CLIENT_IP_SOURCE")
	if value == "" {
		if environment.IsDevelopment() || environment == EnvironmentTest {
			return parsed[ClientIPSource]{value: ClientIPSourceDirect}
		}
		return parsed[ClientIPSource]{err: required("WGA_CLIENT_IP_SOURCE")}
	}

	switch ClientIPSource(value) {
	case ClientIPSourceDirect, ClientIPSourceRailway:
		return parsed[ClientIPSource]{value: ClientIPSource(value)}
	default:
		return parsed[ClientIPSource]{err: fmt.Errorf("WGA_CLIENT_IP_SOURCE must be direct or railway")}
	}
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
func parsePostcards(lookup Lookup, publicURL PublicURL, sender MailSender, tokenKeyring PostcardTokenKeyring) parsed[Postcards] {
	expression := lookup("WGA_POSTCARD_FREQUENCY")
	if expression == "" {
		expression = "*/1 * * * *"
	}

	schedule, scheduleErr := cron.ParseStandard(expression)
	if scheduleErr != nil {
		scheduleErr = fmt.Errorf("WGA_POSTCARD_FREQUENCY must be a valid cron expression")
	}

	return parsed[Postcards]{
		value: Postcards{
			expression:   expression,
			schedule:     schedule,
			tokenKeyring: tokenKeyring,
			Sender:       sender,
			PublicURL:    publicURL,
		},
		err: scheduleErr,
	}
}

// parsePostcardTokenKeyring validates the versioned postcard token keys.
func parsePostcardTokenKeyring(encodedKeys Secret, activeKeyID string) parsed[PostcardTokenKeyring] {
	var errs []error
	if encodedKeys.Value() == "" {
		errs = append(errs, required("WGA_POSTCARD_TOKEN_KEYS"))
	}
	if activeKeyID == "" {
		errs = append(errs, required("WGA_POSTCARD_TOKEN_ACTIVE_KEY_ID"))
	}

	activeKeyIDValid := activeKeyID == "" || validPostcardTokenKeyID(activeKeyID)
	if !activeKeyIDValid {
		errs = append(errs, fmt.Errorf("WGA_POSTCARD_TOKEN_ACTIVE_KEY_ID must be a valid key ID"))
	}

	keyring := PostcardTokenKeyring{
		activeKeyID: activeKeyID,
		keys:        make(map[string]PostcardTokenKey),
	}
	if encodedKeys.Value() == "" {
		return parsed[PostcardTokenKeyring]{value: keyring, err: errors.Join(errs...)}
	}

	var values map[string]string
	if err := json.Unmarshal([]byte(encodedKeys.Value()), &values); err != nil {
		errs = append(errs, fmt.Errorf("WGA_POSTCARD_TOKEN_KEYS must be a JSON object of Base64URL-encoded keys"))
		return parsed[PostcardTokenKeyring]{value: keyring, err: errors.Join(errs...)}
	}
	if len(values) == 0 {
		errs = append(errs, fmt.Errorf("WGA_POSTCARD_TOKEN_KEYS must contain at least one key"))
		return parsed[PostcardTokenKeyring]{value: keyring, err: errors.Join(errs...)}
	}

	keysValid := true
	for keyID, encodedKey := range values {
		if !validPostcardTokenKeyID(keyID) {
			errs = append(errs, fmt.Errorf("WGA_POSTCARD_TOKEN_KEYS contains an invalid key ID"))
			keysValid = false
			break
		}

		decodedKey, err := decodePostcardTokenKey(encodedKey)
		if err != nil {
			errs = append(errs, err)
			keysValid = false
			break
		}
		keyring.keys[keyID] = decodedKey
	}

	if keysValid && activeKeyID != "" && activeKeyIDValid {
		if _, ok := keyring.keys[activeKeyID]; !ok {
			errs = append(errs, fmt.Errorf("WGA_POSTCARD_TOKEN_ACTIVE_KEY_ID must identify a key in WGA_POSTCARD_TOKEN_KEYS"))
		}
	}

	return parsed[PostcardTokenKeyring]{value: keyring, err: errors.Join(errs...)}
}

func validPostcardTokenKeyID(keyID string) bool {
	if keyID == "" || len(keyID) > 64 {
		return false
	}

	for index := 0; index < len(keyID); index++ {
		character := keyID[index]
		if character >= 'a' && character <= 'z' {
			continue
		}
		if character >= 'A' && character <= 'Z' {
			continue
		}
		if character >= '0' && character <= '9' {
			continue
		}
		if character == '-' || character == '_' {
			continue
		}
		return false
	}

	return true
}

func decodePostcardTokenKey(encodedKey string) (PostcardTokenKey, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(encodedKey)
	if err != nil {
		decoded, err = base64.URLEncoding.DecodeString(encodedKey)
	}
	if err != nil {
		return PostcardTokenKey{}, fmt.Errorf("WGA_POSTCARD_TOKEN_KEYS must contain valid Base64URL-encoded keys")
	}
	if len(decoded) != 32 {
		return PostcardTokenKey{}, fmt.Errorf("WGA_POSTCARD_TOKEN_KEYS must contain 32-byte AES-256 keys")
	}

	var key PostcardTokenKey
	copy(key.value[:], decoded)
	return key, nil
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
