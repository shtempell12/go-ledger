package main

import (
	"log"
	"net/http"
	"time"

	"github.com/shtempell12/go-ledger/gateway/internal/api"
	"github.com/shtempell12/go-ledger/ledger"
)

const addr = ":8080"

func main() {
	if err := ledger.Init(); err != nil {
		log.Fatalf("gateway: %v", err)
	}
	defer ledger.Close()

	server := &http.Server{
		Addr:              addr,
		Handler:           api.NewRouter(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("gateway: listening on %s", addr)
	if err := server.ListenAndServe(); err != nil {
		log.Printf("gateway: %v", err)
	}
}
