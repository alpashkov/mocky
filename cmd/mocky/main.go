package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"path/filepath"

	"mocky/internal/config"
	"mocky/internal/daemon"
	"mocky/internal/server"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to YAML config")
	daemonMode := flag.Bool("daemon", false, "run server in background")
	flag.Usage = func() {
		_, _ = fmt.Fprint(flag.CommandLine.Output(), config.HelpText)
	}
	flag.Parse()

	if *daemonMode {
		if err := daemon.Relaunch(); err != nil {
			log.Fatalf("start daemon: %v", err)
		}
		return
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	address := cfg.Server.Address
	if address == "" {
		address = ":8080"
	}

	handler, err := server.New(cfg, filepath.Dir(*configPath))
	if err != nil {
		log.Fatalf("build server: %v", err)
	}

	log.Printf("listening on %s", address)
	if err := http.ListenAndServe(address, handler); err != nil {
		log.Fatalf("http server stopped: %v", err)
	}
}
