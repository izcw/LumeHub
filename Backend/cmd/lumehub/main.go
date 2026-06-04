package main

import (
	"log"
	"net/http"

	"lumehub/internal/config"
	"lumehub/internal/server"
	"lumehub/internal/startup"
)

func main() {
	addr := config.ListenAddr()
	mux := server.NewMux()
	startup.PrintBanner(addr, config.DataDir(), config.WWWDir(), config.AuthPassword() != "")
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
