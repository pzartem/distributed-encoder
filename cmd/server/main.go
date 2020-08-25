package main

import (
	"context"
	"log"
	"net/http"
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
	if err := realMain(); err != nil {
		log.Fatal(err)
	}
}

func realMain() error {
	var cfg EnvConfig
	if err := envconfig.Process(context.Background(), &cfg); err != nil {
		log.Fatalf("Can't log env config: %s", err)
		return err
	}

	srv, err := server.New(server.Config{
		DispatchTimeout: 30 * time.Second,
		Store: &server.FSObjectStore{
			Path: cfg.ResultPath,
		},
		TileStreamer: &transcoder.Transcoder{},
	})
	if err != nil {
		log.Fatalf("Can't start server service: %s", err)
		return err
	}
	defer srv.Close()

	workHandler := handler{s: srv}

	router := httprouter.New()

	router.HandlerFunc(http.MethodPost, "/work/jobs", workHandler.dispatch)
	router.HandlerFunc(http.MethodPost, "/work/result", workHandler.acceptResult)
	router.HandlerFunc(http.MethodPost, "/work/trigger", workHandler.trigger)

	log.Println("HTTP Server started on addr: ", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, router); err != nil {
		return err
	}

	return nil
}
