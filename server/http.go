package server

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
	"mime"
	"net/http"

	"distributed-encoder/worker"
)

type Service interface {
	Dispatch() (*worker.Job, error)
	AcceptResult(string, io.Reader) error
	TriggerWork(EncodeVideoRequest) error
}

type HTTPHandler struct {
	Service Service
}

// POST /work/
func (h HTTPHandler) Dispatch(w http.ResponseWriter, req *http.Request) {
	job, err := h.Service.Dispatch()
	if err == ErrDispatchTimeout {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	if err != nil {
		logErr(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer job.Src.Close()

	log.Println("[HTTP] starting stream")
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
func (h HTTPHandler) AcceptResult(w http.ResponseWriter, req *http.Request) {
	log.Println("[HTTP] accepting result")
	content := req.Header.Get("Content-Disposition")
	_, params, err := mime.ParseMediaType(content)
	if err != nil {
		return
	}
	fileName := params["filename"]
	defer req.Body.Close()

	err = h.Service.AcceptResult(fileName, req.Body)
	if err != nil {
		logErr(err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// POST /work/result
func (h HTTPHandler) Trigger(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()

	var encoderReq EncodeVideoRequest
	dec := json.NewDecoder(req.Body)
	err := dec.Decode(&encoderReq)
	if err != nil {
		logErr(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = h.Service.TriggerWork(encoderReq)
	if err != nil {
		logErr(err)
		writeError(w, err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func writeError(w io.Writer, msg string) {
	b, err := json.Marshal(map[string]interface{}{
		"error": msg,
	})
	if err != nil {
		return
	}

	if _, err = w.Write(b); err != nil {
		logErr(err)
	}
}

func logErr(err error) {
	log.Printf("[HTTP] request error: %s ", err)
}
