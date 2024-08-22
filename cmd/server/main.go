package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/c12s/agent_queue/internal/servers"
	"github.com/nats-io/nats.go"
)

func createNatsConn() (*nats.Conn, error) {
	address := os.Getenv("NATS_CONN_ADDRESS")
	if address == "" {
		log.Panicf("Environment variable 'NATS_CONN_ADDRESS' empty!")
	}
	log.Printf("Trying to connecto to nats on address %s", address)
	return nats.Connect(fmt.Sprintf("nats://%s", address))
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	natsConn, err := createNatsConn()
	if err != nil {
		log.Panicf("Failed to connect to nats, shutting down: %v", err)
	}

	port, err := strconv.Atoi(os.Getenv("GRPC_PORT"))
	if err != nil {
		log.Panicf("Value in 'GRPC_PORT' env variable is not a valid port number")
	}

	servers.Serve(natsConn, port)
}
