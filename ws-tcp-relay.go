package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"golang.org/x/net/websocket"
)

var (
	connectPrefix = []byte("CONNECT ")
	tcpAddress    string
	debug         bool
	auth          string
	log           *zap.Logger
)

func watch(dst io.Writer, src io.Reader, doneCh chan<- bool) {
	var (
		authenticated bool
		err           error
		buf           = make([]byte, 32*1024)
	)

	for {
		nr, _ := src.Read(buf)

		if !authenticated {
			if bytes.HasPrefix(buf, connectPrefix) {
				cj := bytes.TrimPrefix(buf[0:nr], connectPrefix)

				var data map[string]interface{}
				err = json.Unmarshal(cj, &data)
				if err != nil {
					log.Error("json error", zap.Error(err))
				}

				if _, ok := data["auth_token"].(string); !ok {
					doneCh <- true
					return
				}

				req, err := http.NewRequest("GET", auth, nil)
				if err != nil {
					log.Error("req err", zap.Error(err))
					return
				}

				req.Header.Add("Authorization", "Bearer "+data["auth_token"].(string))
				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					log.Error("resp err", zap.Error(err))
					return
				}

				ar := &AuthResponse{}
				err = json.NewDecoder(resp.Body).Decode(&ar)
				if err != nil {
					log.Error("js err", zap.Error(err))
					return
				}

				if !ar.Authorized {
					log.Debug("Client Not Authorized")
					doneCh <- true
					return
				}

				log.Debug("Client Authorized")
				authenticated = true
			}
		}

		if authenticated {
			go copyWorker(dst, src, doneCh)
			return
		}
	}
}

func copyWorker(dst io.Writer, src io.Reader, doneCh chan<- bool) {
	io.Copy(dst, src)
	doneCh <- true
}

func relayHandler(ws *websocket.Conn) {
	log.Debug("Connecting to nats...")
	conn, err := net.Dial("tcp", tcpAddress)
	if err != nil {
		log.Error("[ERROR] TCP Dial", zap.Error(err))
		return
	}

	log.Debug("Connected!")
	doneCh := make(chan bool)

	if auth != "" {
		log.Debug("Waiting for connection info from nats...")
		go watch(conn, ws, doneCh)

	} else {
		log.Debug("No Auth Detected")
		go copyWorker(conn, ws, doneCh)
	}

	go copyWorker(ws, conn, doneCh)

	<-doneCh
	conn.Close()
	ws.Close()
	<-doneCh

	close(doneCh)
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s <tcpTargetAddress>\n", os.Args[0])
	flag.PrintDefaults()
}

type AuthResponse struct {
	Authorized bool `json:"authorized"`
}

func main() {
	var port int
	var certFile string
	var keyFile string

	flag.IntVar(&port, "p", 4223, "Port to listen on.")
	flag.IntVar(&port, "port", 4223, "Port to listen on.")
	flag.StringVar(&certFile, "tlscert", "", "TLS cert file path")
	flag.StringVar(&keyFile, "tlskey", "", "TLS key file path")
	flag.StringVar(&auth, "auth", "", "JWT Authentication url")
	flag.BoolVar(&debug, "debug", false, "Enable logs")
	flag.Usage = usage
	flag.Parse()

	var err error
	if debug {
		config := zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		log, err = config.Build()
		if err != nil {
			panic(err)
		}
	} else {
		log, err = zap.NewProduction()
		if err != nil {
			panic(err)
		}
	}

	tcpAddress = flag.Arg(0)
	if tcpAddress == "" {
		fmt.Fprintln(os.Stderr, "no address specified")
		os.Exit(1)
	}

	portString := fmt.Sprintf(":%d", port)

	log.Info("Starting server on port", zap.Int("port", port))
	http.Handle("/", websocket.Handler(relayHandler))

	if certFile != "" && keyFile != "" {
		err = http.ListenAndServeTLS(portString, certFile, keyFile, nil)
	} else {
		err = http.ListenAndServe(portString, nil)
	}
	if err != nil {
		panic(err)
	}
}
