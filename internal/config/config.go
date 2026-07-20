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

type Lookup func(string) string

type parsed[T any] struct {
	value T
	err   error
}

type Secret struct {
	value string
}

func (s Secret) Value() string {
	return s.value
}

func (Secret) String() string {
	return "[redacted]"
}

func (Secret) GoString() string {
	return "config.Secret([redacted])"
}

type Environment string

const (
	EnvironmentDevelopment Environment = "development"
	EnvironmentTest        Environment = "test"
	EnvironmentStaging     Environment = "staging"
	EnvironmentProduction  Environment = "production"
)

func (e Environment) IsDevelopment() bool {
	return e == EnvironmentDevelopment
}

func (e Environment) AllowsCaptchaBypass() bool {
	return e == EnvironmentDevelopment || e == EnvironmentTest
}

type PublicURL struct {
	url url.URL
}

func (u PublicURL) String() string {
	return u.url.String()
}

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

type Storage struct {
	Endpoint       url.URL
	Bucket         string
	Region         string
	AccessKey      string
	AccessSecret   Secret
	ForcePathStyle bool
}

type MailSender struct {
	Name    string
	Address mail.Address
}

type SMTP struct {
	Host     string
	Port     int
	Username string
	Password Secret
}

type Mail struct {
	Sender MailSender
	SMTP   SMTP
}

type Administrator struct {
	Email    mail.Address
	Password Secret
	Enabled  bool
}

type Captcha struct {
	secret  Secret
	siteKey string
	verify  bool
}

func (c Captcha) Verify() bool {
	return c.verify
}

func (c Captcha) Secret() string {
	return c.secret.Value()
}

func (c Captcha) SiteKey() string {
	return c.siteKey
}

type Postcards struct {
	expression string
	schedule   cron.Schedule
	Sender     MailSender
	PublicURL  PublicURL
}

func (p Postcards) Expression() string {
	return p.expression
}

type Sitemap struct {
	Environment Environment
	PublicURL   PublicURL
}

type Server struct {
	Environment Environment
	PublicURL   PublicURL
	Postcards   Postcards
	Captcha     Captcha
}

func (s Server) Sitemap() Sitemap {
	return Sitemap{
		Environment: s.Environment,
		PublicURL:   s.PublicURL,
	}
}

type InitialSettings struct {
	PublicURL PublicURL
	Storage   Storage
	Mail      Mail
}

type Migrations struct {
	publicURL     parsed[PublicURL]
	storage       parsed[Storage]
	mail          parsed[Mail]
	administrator parsed[Administrator]
}

func (m Migrations) InitialSettings() (InitialSettings, error) {
	settings := InitialSettings{
		PublicURL: m.publicURL.value,
		Storage:   m.storage.value,
		Mail:      m.mail.value,
	}

	return settings, errors.Join(
		m.publicURL.err,
		m.storage.err,
		m.mail.err,
		requireMigrationMail(settings.Mail),
	)
}

func (m Migrations) Administrator() (Administrator, error) {
	return m.administrator.value, m.administrator.err
}

type Config struct {
	environment parsed[Environment]
	publicURL   parsed[PublicURL]
	sender      parsed[MailSender]
	postcards   parsed[Postcards]
	captcha     Captcha
	migrations  Migrations
}

func Load() (Config, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, errors.New("unable to load .env")
	}

	return LoadFrom(os.Getenv), nil
}

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
		migrations: Migrations{
			publicURL:     publicURL,
			storage:       storage,
			mail:          mailConfig,
			administrator: administrator,
		},
	}
}

func (c Config) Environment() Environment {
	return c.environment.value
}

func (c Config) Server() (Server, error) {
	server := Server{
		Environment: c.environment.value,
		PublicURL:   c.publicURL.value,
		Postcards:   c.postcards.value,
		Captcha:     c.captcha,
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

func (c Config) Sitemap() (Sitemap, error) {
	return Sitemap{
		Environment: c.environment.value,
		PublicURL:   c.publicURL.value,
	}, errors.Join(c.environment.err, c.publicURL.err)
}

func (c Config) Migrations() Migrations {
	return c.migrations
}

func parseEnvironment(value string) parsed[Environment] {
	switch Environment(value) {
	case EnvironmentDevelopment, EnvironmentTest, EnvironmentStaging, EnvironmentProduction:
		return parsed[Environment]{value: Environment(value)}
	default:
		return parsed[Environment]{err: fmt.Errorf("WGA_ENV must be one of development, test, staging, production")}
	}
}

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

func required(name string) error {
	return fmt.Errorf("%s must be set", name)
}

func requireValue(name string, value string) error {
	if value == "" {
		return required(name)
	}

	return nil
}

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
