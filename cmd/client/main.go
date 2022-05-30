package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/sethvargo/go-envconfig"

	"distributed-encoder/transcoder"
	"distributed-encoder/worker"
)

type EnvConfig struct {
	ServerAddr string `env:"SERVER_ADDR,default=http://localhost:1111"`
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
		return err
	}

	client := worker.NewClient(
		cfg.ServerAddr+"/work/jobs",
		cfg.ServerAddr+"/work/result",
	)

	w, err := worker.New(client, transcoder.New())
	if err != nil {
		return err
	}

	log.Println("Starting client")
	if err := w.Start(ctx); err != nil {
		return err
	}
	return nil
}
