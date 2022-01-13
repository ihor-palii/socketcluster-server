package main

import (
	"github.com/evalphobia/logrus_sentry"
	"github.com/greatnonprofits-nfp/ccl-chatbot/server/v2/webchat"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	server "github.com/greatnonprofits-nfp/ccl-chatbot/server/v2"
)

func main() {
	serverStartTime := time.Now()
	config := server.LoadConfig("chatbot_server.toml")

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

	// start hub to be able to receive msgs from courier
	hub := webchat.NewHub()
	go hub.Run()

	// run server with main routes
	s := server.NewServer(config)
	s.Router().Get("/", webchat.Index)
	s.Router().Post("/", func(w http.ResponseWriter, r *http.Request) { webchat.MessageReceived(hub, w, r) })
	s.Router().Get("/ping", func(w http.ResponseWriter, r *http.Request) { webchat.Ping(serverStartTime, w, r) })
	s.Router().Get("/socketcluster", func(w http.ResponseWriter, r *http.Request) { webchat.ServeWS(hub, w, r) })
	err = s.Start()
	if err != nil {
		logrus.Fatalf("Error starting server: %s", err)
	}

	// stop server on signal received
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	logrus.WithField("comp", "main").WithField("signal", <-ch).Info("stopping")
	s.Stop()
}
