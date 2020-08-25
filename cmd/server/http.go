package main

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
	"mime"
	"net/http"

	"distributed-encoder/server"
	"distributed-encoder/worker"
)

type handler struct {
	s *server.Server
}

// POST /work/
func (h handler) dispatch(w http.ResponseWriter, req *http.Request) {
	job, err := h.s.Dispatch()
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
	worker.MarshalJobToHeader(job, w.Header())

	if _, err := io.Copy(w, bufio.NewReader(job.Src)); err != nil {
		log.Printf("Serving stream error: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// POST /work/result
func (h handler) acceptResult(w http.ResponseWriter, req *http.Request) {
	log.Println("Result stream started")
	content := req.Header.Get("Content-Disposition")
	_, params, err := mime.ParseMediaType(content)
	if err != nil {
		return
	}
	fileName := params["filename"]
	defer req.Body.Close()

	err = h.s.AcceptResult(fileName, req.Body)
	if err != nil {
		return
	}

	w.WriteHeader(http.StatusOK)
}

// POST /work/result
func (h handler) trigger(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()

	var encoderReq server.EncodeVideoRequest
	dec := json.NewDecoder(req.Body)
	err := dec.Decode(&encoderReq)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = h.s.TriggerWork(encoderReq)
	if err != nil {
		log.Println(err)
		writeError(w, err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func writeError(w http.ResponseWriter, msg string) {
	b, err := json.Marshal(map[string]interface{}{
		"error": msg,
	})
	if err != nil {
		return
	}

	if _, err = w.Write(b); err != nil {
		log.Println(err)
	}
}
