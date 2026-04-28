//go:build windows

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/s5i/goutil/version"
	"github.com/s5i/tassist/exp"
	"github.com/s5i/tassist/server"
	"golang.org/x/sync/errgroup"
)

var (
	accPath = flag.String("acc_path", filepath.Join(os.Getenv("AppData"), "TAssistant", "accounts.yaml"), "Path to accounts file.")
	logPath = flag.String("log_path", filepath.Join(os.Getenv("Temp"), "tassist.log"), "Path to log file.")
)

func main() {
	if err := mainErr(); err != nil {
		log.Printf("Quitting with error: %v", err)
		os.Exit(1)
	}
	log.Printf("Quitting.")
}

func mainErr() (retErr error) {
	flag.Parse()

	if f, err := os.OpenFile(*logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err == nil {
		log.SetOutput(io.MultiWriter(f, os.Stderr))
		defer f.Close()
	}

	defer func() {
		if r := recover(); r != nil {
			log.Printf("Crash detected: %v", r)
			retErr = fmt.Errorf("recover(): %v", r)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ver := version.Get()
	log.Printf("Running Tibiantis Assist %s", ver)

	expCache, err := exp.NewCache()
	if err != nil {
		log.Printf("exp.NewCache() failed: %v", err)
		return err
	}

	srv, err := server.New(*accPath, expCache, ver)
	if err != nil {
		log.Printf("server.New() failed: %v", err)
		return err
	}

	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return expCache.Run(ctx)
	})

	eg.Go(func() error {
		return srv.Run(ctx)
	})

	if err := eg.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}

	return nil
}
