//go:build windows

package main

import (
	"context"
	"errors"
	"flag"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/s5i/goutil/version"
	"github.com/s5i/tassist/acc"
	"github.com/s5i/tassist/exp"
	"github.com/s5i/tassist/server"
	"golang.org/x/sync/errgroup"
)

var (
	dir    = flag.String("dir", filepath.Join(os.Getenv("AppData"), "TAssistant"), "Path to persistent dir.")
	tmpDir = flag.String("tmp_dir", filepath.Join(os.Getenv("Temp"), "tassist"), "Path to temp dir.")
)

func main() {
	flag.Parse()

	if err := os.MkdirAll(*tmpDir, 0755); err != nil {
		log.Printf("Quitting with error: %v", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(*dir, 0755); err != nil {
		log.Printf("Quitting with error: %v", err)
		os.Exit(1)
	}

	if f, err := os.OpenFile(filepath.Join(*tmpDir, "tassist.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err == nil {
		log.SetOutput(io.MultiWriter(f, os.Stderr))
		defer f.Close()
	}

	if err := mainErr(); err != nil {
		log.Printf("Quitting with error: %v", err)
		os.Exit(1)
	}

	log.Printf("Quitting.")
}

func mainErr() (retErr error) {
	defer logPanic()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ver := version.Get()
	log.Printf("Running Tibiantis Assist %s", ver)

	st, err := acc.New(*dir)
	if err != nil {
		return err
	}

	expCache, err := exp.NewCache(*tmpDir)
	if err != nil {
		log.Printf("exp.NewCache() failed: %v", err)
		return err
	}

	srv, err := server.New(st, expCache, ver)
	if err != nil {
		log.Printf("server.New() failed: %v", err)
		return err
	}

	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		defer logPanic()
		return expCache.Run(ctx)
	})

	eg.Go(func() error {
		defer logPanic()
		return srv.Run(ctx)
	})

	if err := eg.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}

	return nil
}

func logPanic() {
	if r := recover(); r != nil {
		log.Printf("Crash detected: %v", r)
	}
}
