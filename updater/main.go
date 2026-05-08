package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"
)

func main() {
	log.SetFlags(0)
	if mainErr() {
		log.Printf("\nUpdate successful, restarting...")
		go exec.Command("cmd", "/C", "start", os.Args[1]).Run()
		time.Sleep(3 * time.Second)
	} else {
		log.Printf("\n[press Enter to exit]")
		fmt.Scanln()
	}
}

func mainErr() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if len(os.Args) < 3 {
		log.Printf("Usage: %s [target_exe] [new_exe]", os.Args[0])
		return false
	}

	log.Printf("%s %q %q\n\n", os.Args[0], os.Args[1], os.Args[2])

	dest := os.Args[1]
	src := os.Args[2]

	log.Printf("Waiting for %s to become writeable...", dest)

	for {
		select {
		case <-ctx.Done():
			log.Printf("Failed to overwrite %s.", dest)
			return false
		case <-time.After(time.Second):
			if err := os.Rename(src, dest); err == nil {
				log.Printf("Replaced %s with %s.", dest, src)
				return true
			}
		}
	}
}
