package ccl_chatbot_server

import "github.com/nyaruka/ezconf"

// Config is our top level configuration object
type Config struct {
	Domain    string `help:"the domain courier is exposed on"`
	Address   string `help:"the network interface address courier will bind to"`
	Port      int    `help:"the port courier will listen on"`
	SentryDSN string `help:"the DSN used for logging errors to Sentry"`
	LogLevel  string `help:"the logging level courier should use"`
	Version   string `help:"the version that will be used in request and response headers"`
}

// NewConfig returns a new default configuration object
func NewConfig() *Config {
	return &Config{
		Domain:   "rapidpro",
		Address:  "",
		Port:     9090,
		LogLevel: "debug",
		Version:  "Dev",
	}
}

// LoadConfig loads our configuration from the passed in filename
func LoadConfig(filename string) *Config {
	config := NewConfig()
	loader := ezconf.NewLoader(
		config,
		"chatbot-server", "Chatbot Server - A fast message broker for WebSocket messages",
		[]string{filename},
	)

	loader.MustLoad()
	return config
}
