package main

import (
	"context"
	"log"

	"github.com/sethvargo/go-envconfig"

	"distributed-encoder/transcoder"
	"distributed-encoder/worker"
)

type EnvConfig struct {
	ServerAddr string `env:"SERVER_ADDR,default=http://localhost:1111"`
}

func main() {
	if err := realMain(); err != nil {
		log.Fatal(err)
	}
}

func realMain() error {
	var cfg EnvConfig
	if err := envconfig.Process(context.Background(), &cfg); err != nil {
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
	if err := w.Start(context.Background()); err != nil {
		return err
	}
	return nil
}
