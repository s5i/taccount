//go:build windows

package main

import (
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"

	"github.com/s5i/taccount/server"
	"github.com/s5i/taccount/storage"
)

func main() {
	yamlPath := yamlFilePath()

	entries, err := storage.Load(yamlPath)
	if err != nil {
		log.Fatalf("Failed to load accounts: %v", err)
	}

	srv := server.New(&entries, yamlPath)

	url, err := srv.ListenAndServe()
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	log.Printf("Listening on %s", url)

	// Open the browser.
	exec.Command("cmd", "/c", "start", url).Start()

	// Wait for interrupt.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig
}

func yamlFilePath() string {
	exe, err := os.Executable()
	if err != nil {
		return "accounts.yaml"
	}
	return filepath.Join(filepath.Dir(exe), "accounts.yaml")
}
