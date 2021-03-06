package cmd

import (
	"context"
	// "net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	// "github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	// "github.com/fancar/wrenches/internal/config"
)

func run(cnd *cobra.Command, args []string) error {
	setLogLevel()
	tasks := []func(context.Context, *sync.WaitGroup) error{
		// setLogLevel,
		printStartMessage,
		// setupPrometheus,
		// startSomeRoutine,
	}

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())

	for _, t := range tasks {
		if err := t(ctx, &wg); err != nil {
			log.Fatal(err)
		}
	}

	exitChan := make(chan struct{})
	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan
	go func() {
		cancel()
		wg.Wait()
		exitChan <- struct{}{}
	}()
	cancel()
	select {
	case <-exitChan:
	case s := <-sigChan:
		log.WithField("signal", s).Info("signal received, terminating")
	}

	return nil
}

func printStartMessage(ctx context.Context, wg *sync.WaitGroup) error {
	log.WithFields(log.Fields{
		"version": version,
		// "docs":    "https://www. ... .su/",
	}).Info("starting iot-tools ...")
	return nil
}

// func setupPrometheus(ctx context.Context, wg *sync.WaitGroup) error {
// 	log.WithFields(log.Fields{
// 		"bind": config.C.Prometheus.Bind,
// 	}).Info("starting Prometheus endpoint server")

// 	mux := http.NewServeMux()
// 	mux.Handle("/metrics", promhttp.Handler())

// 	server := http.Server{
// 		Handler: mux,
// 		Addr:    config.C.Prometheus.Bind,
// 	}

// 	go func() {
// 		err := server.ListenAndServe()
// 		log.WithError(err).Error("prometheus endpoint server error")
// 	}()

// 	return nil
// }
