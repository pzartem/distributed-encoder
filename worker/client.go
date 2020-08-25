package worker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"
)

// HandleJobFunc is triggered when job is called
type HandleJobFunc func(*Job) error

const (
	defaultRetryTimeout = 5 * time.Second
)

// HTTPClient connects to server and gets jobs using long polling
type HTTPClient struct {
	client *http.Client

	pollEndpoint   string
	resultEndpoint string
}

// NewClient creates new HTTPClient
func NewClient(pollEndpoint, resultEndpoint string) *HTTPClient {
	return &HTTPClient{
		client:         &http.Client{},
		pollEndpoint:   pollEndpoint,
		resultEndpoint: resultEndpoint,
	}
}

var (
	// ErrCancelled happen when polling is canceled
	ErrCancelled = errors.New("canceled")
)

// Subscribe subscribes for the jobs
func (c *HTTPClient) Subscribe(ctx context.Context, handlerFunc HandleJobFunc) error {
	for {
		select {
		case <-ctx.Done():
			log.Println("[poll] canceled")
			return ErrCancelled
		default:
			if err := c.pollingFlow(handlerFunc); err != nil {
				log.Printf("[poll] error: %s, retry timeout 5 sec", err)
				time.Sleep(defaultRetryTimeout)
				return nil
			}
		}
	}
}

func (c *HTTPClient) pollingFlow(handler HandleJobFunc) error {
	log.Println("[poll] start")
	res, err := c.poll()
	if err != nil {
		return err
	}

	switch res.StatusCode {
	case http.StatusNotModified: // timed out try again
		log.Println("[poll] timeout")
		return nil
	case http.StatusOK:
		log.Println("[poll] answered")
		err := handle(res, handler)
		if err != nil {
			return err
		}
	default:
		log.Println("[poll] unexpected status code: ", res.StatusCode)
	}

	return nil
}

func handle(res *http.Response, handlerFunc HandleJobFunc) error {
	job, err := ParseJobFromHTTP(res)
	if err != nil {
		return err
	}
	if err := handlerFunc(&job); err != nil {
		return err
	}

	return nil
}

func (c *HTTPClient) poll() (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, c.pollEndpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Connection", "keep-alive")

	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// SendResult send result to server
func (c *HTTPClient) SendResult(fileName string, body io.Reader) error {
	req, err := http.NewRequest(http.MethodPost, c.resultEndpoint, body)
	if err != nil {
		return err
	}
	defer req.Body.Close()

	header := req.Header
	header.Set("Content-Disposition", "attachment; filename="+fileName)
	header.Set("Content-Type", "application/octet-stream")

	res, err := c.client.Do(req)
	if err != nil {
		return err
	}
	res.Body.Close()

	return nil
}

const (
	tileHeader   = "X-Tile"
	heightHeader = "X-Height"
	widthHeader  = "X-Width"
)

// ParseJobFromHTTP parses worker Job from http.Response
func ParseJobFromHTTP(res *http.Response) (Job, error) {
	h := res.Header
	tileName := h.Get(tileHeader)
	if tileName == "" {
		return Job{}, fmt.Errorf(tileHeader + " is invalid")
	}
	height, err := strconv.Atoi(h.Get(heightHeader))
	if err != nil {
		return Job{}, err
	}
	width, err := strconv.Atoi(h.Get(widthHeader))
	if err != nil {
		return Job{}, err
	}

	return Job{
		TileName: tileName,
		Height:   height,
		Width:    width,
		Src:      res.Body,
	}, nil
}

func MarshalJobToHeader(job *Job, header http.Header) {
	header.Set(tileHeader, job.TileName)
	header.Set(heightHeader, strconv.Itoa(job.Height))
	header.Set(widthHeader, strconv.Itoa(job.Width))
}
