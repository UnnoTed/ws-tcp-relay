# ws-tcp-relay
[![License MIT](https://img.shields.io/npm/l/express.svg)](http://opensource.org/licenses/MIT)

A relay between Websocket and TCP. All messages will be copied from all 
Websocket connections to the target TCP server, and vice-versa.

In other words, it's [websocketd](https://github.com/joewalnes/websocketd), but for TCP connections instead of `STDIN` and `STDOUT`.

## Installation
```go get -u github.com/joshglendenning/ws-tcp-relay```

## Usage
```
Usage: ws-tcp-relay <tcpTargetAddress>
  -p int
        Port to listen on. (default 1337)
  -port int
        Port to listen on. (default 1337)
  -tlscert string
        TLS cert file path
  -tlskey string
        TLS key file path
  -debug
        Enable logs
  -auth string
        Url for jwt auth, ws-tcp-relay will send a GET request for each client's connection, it should return { "authorized": true } so the client can connect to the server
```

## WSS Support
To use secure websockets simply specify both the `tlscert` and `tlskey` flags.

## Building
`go build ws-tcp-relay.go`
