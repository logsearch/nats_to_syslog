package main

import (
	"encoding/json"
	"flag"
	"github.com/nats-io/nats"
	"github.com/pivotal-golang/lager"
	"log/syslog"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

var stop chan bool
var logger lager.Logger

func main() {
	logger = lager.NewLogger("nats-to-syslog")

	stop = make(chan bool)
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT)
	signal.Notify(signals, syscall.SIGKILL)
	signal.Notify(signals, syscall.SIGTERM)

	go func() {
		for signal := range signals {
			logger.Info("signal-caught", lager.Data{"signal": signal})
			stop <- true
		}
	}()

	var natsUri = flag.String("nats-uri", "nats://localhost:4222", "The NATS server URI")
	var natsSubject = flag.String("nats-subject", ">", "The NATS subject to subscribe to")
	var syslogEndpoint = flag.String("syslog-endpoint", "localhost:514", "The remote syslog server host:port")
	var debug = flag.Bool("debug", false, "debug logging true/false")
	flag.Parse()

	if *debug {
		logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))
	} else {
		logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.INFO))
	}

	syslog, err := syslog.Dial("tcp", *syslogEndpoint, syslog.LOG_INFO, "nats-to-syslog")
	handleError(err, "connecting to syslog")
	logger.Info("connected-to-syslog", lager.Data{"endpoint": syslogEndpoint})
	defer syslog.Close()

	natsClient, err := nats.Connect(*natsUri)
	handleError(err, "connecting to nats")
	logger.Info("connected-to-nats", lager.Data{"uri": natsUri})
	defer natsClient.Close()

	buffer := make(chan *nats.Msg, 1000)

	go func() {
		for message := range buffer {
			logEntry := buildLogEntry(message)
			logger.Debug("message-sent-to-syslog", lager.Data{"message": logEntry})
			err = syslog.Info(logEntry)
			if err != nil {
				logger.Error("logging-to-syslog-failed", err)
				stop <- true
			}
		}
	}()

	natsClient.Subscribe(*natsSubject, func(message *nats.Msg) {
		buffer <- message
	})
	logger.Info("subscribed-to-subject", lager.Data{"subject": *natsSubject})

	<-stop
	logger.Info("bye.")
}

func handleError(err error, context string) {
	if err != nil {
		context = strings.Replace(context, " ", "-", -1)
		errorLogger := logger.Session(context)
		errorLogger.Error("error", err)
		os.Exit(1)
	}
}

func buildLogEntry(message *nats.Msg) string {
	entry := struct {
		Data    string
		Reply   string
		Subject string
	}{
		string(message.Data),
		message.Reply,
		message.Subject,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		logger.Error("unmarshalling-log-failed", err, lager.Data{"data": string(message.Data)})
		return ""
	}

	return string(data)
}
