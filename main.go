package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/keatontaylor/healthchecker-go/pkg/healthchecker"
)

type urlArrayFlags []string

var (
	healthcheck_interval time.Duration
	urls                 urlArrayFlags
)

func getConfig(fs *flag.FlagSet) []string {
	cfg := make([]string, 0, 10)
	fs.VisitAll(func(f *flag.Flag) {
		cfg = append(cfg, fmt.Sprintf("%s:%q", f.Name, f.Value.String()))
	})
	return cfg
}

func (i *urlArrayFlags) String() string {
	return fmt.Sprintf("%s", *i)
}

func (i *urlArrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func init() {
	flag.DurationVar(
		&healthcheck_interval,
		"interval",
		10*time.Second,
		"Interval for the healthchecks",
	)
	flag.Var(
		&urls,
		"url",
		"URLs to perform health checks against. Can be included multiple times for additonal URLs",
	)

	flag.Parse()
	log.Printf("app.config %v\n", getConfig(flag.CommandLine))
}

func main() {

	// Create context and http server for prom metrics
	ctx, cancel := context.WithCancel(context.Background())

	// Start the collector
	healthchecker := healthchecker.NewHealthChecker(ctx, healthcheck_interval, urls)
	healthchecker.StartCollector()

	// start the http server
	server := &http.Server{Addr: ":2112", Handler: nil}
	go func() {
		err := server.ListenAndServe()
		if err == http.ErrServerClosed {
			log.Printf("http server shutdown/closed: %s\n", err)
		} else if err != nil {
			log.Fatalf("http server stopped with error: %s\n", err)
		} else {
			log.Printf("http server stopped")
		}
	}()

	// Signal to safely shutdown for interrupts and force quit for SIGTERM
	signalChannel := make(chan os.Signal, 2)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)
	sig := <-signalChannel
	switch sig {
	case os.Interrupt:
		log.Println("received interrupt, shutting down.")
		cancel()
		server.Shutdown(context.Background())
	case syscall.SIGTERM:
		log.Println("received sigterm, force quitting.")
		os.Exit(1)
	}

}
