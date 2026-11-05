package main

import (
	"log"
	"net/http"
	"time"

	"github.com/shtempell12/go-ledger/gateway/internal/api"
)

const addr = ":8080"

func main() {
	server := &http.Server{
		Addr:              addr,
		Handler:           api.NewRouter(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("gateway: listening on %s", addr)
	log.Fatal(server.ListenAndServe())
}
