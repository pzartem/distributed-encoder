package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/sethvargo/go-envconfig"

	"distributed-encoder/server"
	"distributed-encoder/transcoder"
)

type EnvConfig struct {
	Addr       string `env:"ADDR,default=:1111"`
	ResultPath string `env:"RESULT_PATH"`

	WorkersAddr []string `env:"WORKERS_ADDR"`
}

func main() {
	ctx, done := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	defer func() {
		done()
		if r := recover(); r != nil {
			log.Fatal("application panic", "panic", r)
		}
	}()

	err := realMain(ctx)
	done()

	if err != nil {
		log.Fatal(err)
	}
	log.Println("successful shutdown")
}

func realMain(ctx context.Context) error {
	var cfg EnvConfig
	if err := envconfig.Process(ctx, &cfg); err != nil {
		log.Fatalf("Can't log env config: %s", err)
		return err
	}

	srv, err := server.New(server.Config{
		DispatchTimeout: 30 * time.Second,
		Store: &server.FSObjectStore{
			Path: cfg.ResultPath,
		},
		TileStreamer: transcoder.New(),
	})
	if err != nil {
		log.Fatalf("Can't start server service: %s", err)
		return err
	}
	defer srv.Close()

	workHandler := server.HTTPHandler{Service: srv}

	router := httprouter.New()

	router.HandlerFunc(http.MethodPost, "/work/jobs", workHandler.Dispatch)
	router.HandlerFunc(http.MethodPost, "/work/result", workHandler.AcceptResult)
	router.HandlerFunc(http.MethodPost, "/work/trigger", workHandler.Trigger)

	log.Println("HTTP Server started on addr: ", cfg.Addr)

	server := http.Server{
		Addr:    cfg.Addr,
		Handler: router,
	}

	// Spawn a goroutine that listens for context closure. When the context is
	// closed, the server is stopped.
	errCh := make(chan error, 1)
	go func() {
		<-ctx.Done()

		log.Println("server.Serve: context closed")
		shutdownCtx, done := context.WithTimeout(context.Background(), 5*time.Second)
		defer done()

		log.Println("server.Serve: shutting down")
		errCh <- server.Shutdown(shutdownCtx)
	}()
	// Run the server. This will block until the provided context is closed.
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("failed to serve: %w", err)
	}

	// Return any errors that happened during shutdown.
	if err := <-errCh; err != nil {
		return fmt.Errorf("failed to shutdown: %w", err)
	}
	return nil
}
