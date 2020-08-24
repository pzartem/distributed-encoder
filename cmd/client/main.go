package main

import (
	"context"
	"log"

	"github.com/sethvargo/go-envconfig"

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
		cfg.ServerAddr+"/work/ack",
		cfg.ServerAddr+"/work/result",
	)

	var encoder worker.VideoEncoder

	w, err := worker.New(client, encoder)
	if err != nil {
		return err
	}

	if err := w.Start(); err != nil {
		return err
	}
	return nil
}