package main

import (
	"github.com/evalphobia/logrus_sentry"
	"github.com/greatnonprofits-nfp/ccl-chatbot/server/v2/handlers"
	"github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"

	webchat "github.com/greatnonprofits-nfp/ccl-chatbot/server/v2"
)

func main() {
	config := webchat.LoadConfig("chatbot_server.toml")

	// configure our logger
	logrus.SetOutput(os.Stdout)
	level, err := logrus.ParseLevel(config.LogLevel)
	if err != nil {
		logrus.Fatalf("Invalid log level '%s'", level)
	}
	logrus.SetLevel(level)

	// if we have a DSN entry, try to initialize it
	if config.SentryDSN != "" {
		hook, err := logrus_sentry.NewSentryHook(config.SentryDSN, []logrus.Level{logrus.PanicLevel, logrus.FatalLevel, logrus.ErrorLevel})
		if err != nil {
			logrus.Fatalf("Invalid sentry DSN: '%s': %s", config.SentryDSN, err)
		}
		hook.Timeout = 0
		hook.StacktraceConfiguration.Enable = true
		hook.StacktraceConfiguration.Skip = 4
		hook.StacktraceConfiguration.Context = 5
		logrus.StandardLogger().Hooks.Add(hook)
	}

	server := webchat.NewServer(config)
	server.Router().Get("/", handlers.IndexHandler)
	server.Router().Post("/", handlers.ReceiveMessageHandler)
	server.Router().Get("/ping", handlers.PingHandler)
	server.Router().Get("/socketcluster", handlers.WebSocketConnectionHandler)
	err = server.Start()
	if err != nil {
		logrus.Fatalf("Error starting server: %s", err)
	}

	// stop server on signal received
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	logrus.WithField("comp", "main").WithField("signal", <-ch).Info("stopping")
	server.Stop()
}
