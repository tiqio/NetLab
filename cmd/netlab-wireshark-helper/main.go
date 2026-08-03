package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/netlab/netlab/internal/clienttools/wiresharkhelper"
)

var (
	version              = "dev"
	defaultAllowedOrigin string
)

func main() {
	address := flag.String("listen", wiresharkhelper.DefaultAddress, "loopback listen address")
	allowedOrigin := flag.String("allow-origin", defaultAllowedOrigin, "trusted NetLab origin, for example http://10.72.1.159:18082")
	wiresharkPath := flag.String("wireshark", "", "optional Wireshark executable path")
	showVersion := flag.Bool("version", false, "print version")
	flag.Parse()
	if *showVersion {
		fmt.Println(version)
		return
	}
	if err := wiresharkhelper.ValidateListenAddress(*address); err != nil {
		fmt.Fprintln(os.Stderr, "invalid listen address:", err)
		os.Exit(2)
	}
	if *allowedOrigin == "" {
		fmt.Fprintln(os.Stderr, "-allow-origin is required")
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	handler := wiresharkhelper.Handler{
		AllowedOrigin: *allowedOrigin,
		Version:       version,
		Launcher:      wiresharkhelper.NewRealLauncher(*wiresharkPath),
	}
	fmt.Printf("NetLab Wireshark helper %s listening on http://%s for %s\n", version, *address, *allowedOrigin)
	if err := wiresharkhelper.Serve(ctx, *address, handler); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
