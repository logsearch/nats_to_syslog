package main

import (
	"flag"
	"github.com/nats-io/nats"
	"log/syslog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	stop := make(chan bool)
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT)
	signal.Notify(signals, syscall.SIGKILL)
	signal.Notify(signals, syscall.SIGTERM)

	go func() {
		for _ = range signals {
			stop <- true
		}
	}()

	var natsUri = flag.String("nats-uri", "nats://localhost:4222", "The NATS server URI")
	var syslogEndpoint = flag.String("syslog-server", "localhost:514", "The remote syslog server host:port")
	flag.Parse()

	logger, err := syslog.Dial("tcp", *syslogEndpoint, syslog.LOG_INFO, "nats-to-syslog")
	handleError(err)
	defer logger.Close()

	natsClient, err := nats.Connect(*natsUri)
	handleError(err)
	defer natsClient.Close()

	buffer := make(chan string, 1000)

	go func() {
		for message := range buffer {
			logger.Info(message)
		}
	}()

	natsClient.Subscribe(">", func(message *nats.Msg) {
		buffer <- string(message.Data)
	})

	<-stop
}

func handleError(err error) {
	if err != nil {
		panic(err)
	}
}
