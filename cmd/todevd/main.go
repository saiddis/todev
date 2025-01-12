package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/rollbar/rollbar-go"
	"github.com/saiddis/todev"
	"github.com/saiddis/todev/http"
	"github.com/saiddis/todev/inmem"
	"github.com/saiddis/todev/postgres"
	"github.com/spf13/viper"
)

const (
	// DefaultConfigPath is the default path to the application configuration.
	DefaultConfigPath = "../../config.yaml"
)

// Build version, injected during build.
var (
	version string
	commint string
)

func main() {
	// Propogate build information to root package to share globally.
	todev.Version = strings.TrimPrefix(version, "")
	todev.Commit = commint

	// Setup signal handlers.
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() { <-c; cancel() }()

	// Initialize a new type to represent our application.
	// This type lets us share setup code with our end-to-end tests.
	m := NewMain()

	// Parse command line args and load configuration.
	if err := m.ParseFlags(ctx, os.Args[1:]); err == flag.ErrHelp {
		os.Exit(1)
	} else if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Execute programm.
	if err := m.Run(ctx); err != nil {
		m.Close()
		fmt.Fprintln(os.Stderr, err)
		todev.ReportError(ctx, err)
		os.Exit(1)
	}

	// Wait for CTRL-C.
	<-ctx.Done()

	// Clean up programm.
	if err := m.Close(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// Main represents the program.
type Main struct {
	// Configuration path and parsed config data.
	Config     Config
	ConfigPath string

	// PostgreSQL database used by PostgresSQl service implementation.
	DB *postgres.Conn

	// HTTP server for handling HTTP communication.
	// PostgresSQL services are attached to it before running.
	HTTPServer *http.Server

	// Services exposed for end-to-end tests.
	UserService todev.UserService
}

func NewMain() *Main {
	dsn, ok := todev.GetFromEnv("../../.env", "DB_URL")
	if !ok {
		log.Fatalf("failed to find config value for: %s", dsn)
	}

	return &Main{
		DB:         postgres.New(dsn),
		HTTPServer: http.NewServer(),
	}
}

// Run executes the program. The configuration should already be set up before
// calling this function.
func (m *Main) Run(ctx context.Context) (err error) {
	// Initialize error tracking
	if m.Config.Rollbar.Token != "" {
		rollbar.SetToken(m.Config.Rollbar.Token)
		rollbar.SetEnvironment("production")
		rollbar.SetCodeVersion(version)
		rollbar.SetServerRoot("github.com/saiddis/todev")
		todev.ReportError = rollbarReportError
		todev.ReportPanic = rollbarReportPanic
		log.Print("rollbar error tracking enabled")
	}

	// Initialize event service for real-time events.
	eventService := inmem.NewEventService()

	m.DB.EventService = eventService

	if m.DB.DSN, err = expand(m.Config.DB.DSN); err != nil {
		return fmt.Errorf("error expanding dsn: %w", err)
	}

	if err = m.DB.Open(); err != nil {
		return fmt.Errorf("error openning db: %w", err)
	}

	// Initialize PostgreSQL-backed services.
	authService := postgres.NewAuthService(m.DB)
	repoService := postgres.NewRepoService(m.DB)
	contributorService := postgres.NewContrubutorService(m.DB)
	taskService := postgres.NewTaskService(m.DB)
	userService := postgres.NewUserService(m.DB)

	// Attach user service Main for testing.
	m.UserService = userService

	// Copy configuration settings to the HTTP server.
	m.HTTPServer.Addr = m.Config.HTTP.Addr
	m.HTTPServer.Domain = m.Config.HTTP.Domain
	m.HTTPServer.HashKey = m.Config.HTTP.HashKey
	m.HTTPServer.BlockKey = m.Config.HTTP.BlockKey
	m.HTTPServer.GitHubClientID = m.Config.Github.ClientID
	m.HTTPServer.GitHubClientSecret = m.Config.Github.ClientSecret

	m.HTTPServer.AuthService = authService
	m.HTTPServer.RepoService = repoService
	m.HTTPServer.ContributorService = contributorService
	m.HTTPServer.UserService = userService
	m.HTTPServer.TaskService = taskService
	m.HTTPServer.EventService = eventService

	// Start HTTP server.
	if err = m.HTTPServer.Open(); err != nil {
		return err
	}

	// If TLS enabled, redirect non-TLS connections to TLS.
	if m.HTTPServer.UseTLS() {
		go func() {
			log.Fatal(http.ListenAndServeTLSRedirect(m.Config.HTTP.Domain))
		}()
	}

	// Enable internal debug endpoints.
	go func() { http.ListenAndServeDebug() }()

	log.Printf("running: url=%q debug=http://localhost:6060 dsn=%q", m.HTTPServer.URL(), m.Config.DB.DSN)

	return nil
}

// Close gracefully closes the program.
func (m *Main) Close() error {
	if m.HTTPServer != nil {
		if err := m.HTTPServer.Close(); err != nil {
			return err
		}
	}

	if m.DB != nil {
		if err := m.DB.Close(); err != nil {
			return err
		}
	}
	return nil
}

// ParseFlags parses the command ling arguments and loads the config.
func (m *Main) ParseFlags(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("todevd", flag.ContinueOnError)
	fs.StringVar(&m.ConfigPath, "config", DefaultConfigPath, "config path")
	if err := fs.Parse(args); err != nil {
		return err
	}

	configPath, err := expand(m.ConfigPath)
	if err != nil {
		return err
	}

	config, err := ReadConfigFile(configPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("config file not found: %v", err)
	} else if err != nil {
		return err
	}

	m.Config = config
	return nil
}

// Config represensts the CLI configuration file.
type Config struct {
	DB struct {
		DSN string `mapstructure:"dsn"`
	} `mapstructure:"db"`

	HTTP struct {
		Addr     string `mapstructure:"addr"`
		Domain   string `mapstructure:"domain"`
		HashKey  string `mapstructure:"hash_key"`
		BlockKey string `mapstructure:"block_key"`
	} `mapstructure:"http"`

	GoogleAnalytics struct {
		MeasurementID string `mapstructure:"measurement_id"`
	} `mapstructure:"google_analytics"`

	Github struct {
		ClientID     string `mapstructure:"client_id"`
		ClientSecret string `mapstructure:"client_secret"`
	} `mapstructure:"github"`

	Rollbar struct {
		Token string `mapstructure:"token"`
	} `mapstructure:"rollbar"`
}

// DefaultConfig returns a new instance of config with defaul set.
func DefaultConfig() Config {
	var config Config
	if dsn, ok := todev.GetFromEnv(todev.DefaultEnvFilePath, "DB_URL"); ok {
		config.DB.DSN = dsn
		return config
	}

	log.Fatal("failed to get value from env file for dsn")
	return config
}

// ReadConfigFile unmarshals configs from a config file.
func ReadConfigFile(filename string) (Config, error) {
	var config Config
	viper.SetConfigFile(filename)

	if err := viper.ReadInConfig(); err != nil {
		return config, err
	} else if err = viper.Unmarshal(&config); err != nil {
		return config, err
	}

	return config, nil
}

// expand returns path using tilde expansion. This means that a file path that
// begins with the "~" will be expanded to prefix the user's home directory.
func expand(path string) (string, error) {
	// Ignore if path has no leading tilde.
	if path != "~" && !strings.HasPrefix(path, "~"+string(os.PathSeparator)) {
		return path, nil
	}

	u, err := user.Current()
	if err != nil {
		return path, err
	} else if u.HomeDir == "" {
		return path, fmt.Errorf("home directory unset")
	}

	if path == "~" {
		return u.HomeDir, nil
	}

	return filepath.Join(u.HomeDir, strings.TrimPrefix(path, "~"+string(os.PathSeparator))), nil

}

// rollbarReportError reports internal errors to rollbar.
func rollbarReportError(ctx context.Context, err error, args ...interface{}) {
	if todev.ErrorCode(err) != todev.EINTERNAL {
		return
	}

	if u := todev.UserFromContext(ctx); u != nil {
		rollbar.SetPerson(fmt.Sprint(u.ID), u.Name, u.Email)
	} else {
		rollbar.ClearPerson()
	}

	log.Printf("error: %v", err)
	rollbar.Error(append([]interface{}{err}, args)...)
}

// rollbarReportPanic reports panics to rollbar.
func rollbarReportPanic(err interface{}) {
	log.Printf("panic: %v", err)
	rollbar.LogPanic(err, true)
}
