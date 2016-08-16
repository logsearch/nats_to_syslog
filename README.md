# nats_to_syslog
Subscribes to NATs message bus and forwards messages to remote Syslog server

# Testing

## Dependencies

```sh
go get github.com/onsi/ginkgo/ginkgo
go get github.com/nats-io/gnatsd

godep go test
```

# Build Instructions

- Ensure you have go 1.6.x installed
- To cross compile for linux on a mac:

```
cd nats_to_syslog/
GOOS=linux GOARCH=amd64 go build
```

- Omit the env vars if building on linux:

`cd nats_to_syslog/ && go build`
