package main

import (
	"bufio"
	"context"
	"io"
	"log"
	"mime"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/sethvargo/go-envconfig"

	"distributed-encoder/server"
	"distributed-encoder/worker"
)

type Config struct {
	Addr       string `env:"ADDR,default=:1111"`
	ResultPath string `env:"RESULT_PATH"`
	InputPath  string `env:"INPUT_PATH"`

	WorkersAddr []string `env:"WORKERS_ADDR"`
}

func main() {
	if err := realMain(); err != nil {
		log.Fatal(err)
	}
}

func realMain() error {
	var cfg Config
	if err := envconfig.Process(context.Background(), &cfg); err != nil {
		log.Fatalf("Can't log env config: %s", err)
		return err
	}
	server, err := server.New(server.Config{
		DispatchTimeout: 30 * time.Second,
	})
	if err != nil {
		log.Fatalf("Can't start server service: %s", err)
		return err
	}

	router := httprouter.New()

	router.HandlerFunc(http.MethodPost, "/work/get", WorkPoll(server))
	router.HandlerFunc(http.MethodPost, "/work/result", WorkResultHandler(server))

	log.Println("HTTP Server started on addr: ", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, router); err != nil {
		log.Fatal(err)
	}

	return nil
}

// POST /work/poll
func WorkPoll(s *server.Server) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		job, err := s.Dispatch()
		if err == server.ErrDispatchTimeout {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer job.Src.Close()

		req.Header.Set("Content-Type", "application/octet-stream")
		worker.MarshalToHeader(job, req.Header)
		if _, err := io.Copy(w, bufio.NewReader(job.Src)); err != nil {
			log.Printf("Serving stream error: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

// POST /work/result
func WorkResultHandler(s *server.Server) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		log.Println("Result stream started")
		content := req.Header.Get("Content-Disposition")
		_, params, err := mime.ParseMediaType(content)
		if err != nil {
			return
		}
		fileName := params["filename"]
		defer req.Body.Close()

		err = s.AcceptResult(fileName, req.Body)
		if err != nil {
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
