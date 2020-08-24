package worker

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
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

// Subscribe subscribes for the jobs
func (c *HTTPClient) Subscribe(handlerFunc HandleJobFunc) error {
	for {
		res, err := c.poll()
		if err != nil {
			log.Println(err)
			continue
		}

		switch res.StatusCode {
		case http.StatusNotModified: // timed out try again
			continue
		case http.StatusOK:
			err := handle(res, handlerFunc)
			if err != nil {
				log.Println(err)
			}
		default:
			log.Println("unexpected status code: ", res.StatusCode)
			continue
		}
	}
}

func handle(res *http.Response, handlerFunc func(Job) error) error {
	job, err := ParseJobFromHTTP(res)
	if err != nil {
		return err
	}
	if err := handlerFunc(job); err != nil {
		return err
	}

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

func MarshalToHeader(job *Job, header http.Header) {
	header.Set(tileHeader, job.TileName)
	header.Set(heightHeader, strconv.Itoa(job.Width))
	header.Set(widthHeader, strconv.Itoa(job.Width))
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

func (c *HTTPClient) SendResult(fileName string, body io.Reader) error {
	req, err := http.NewRequest(http.MethodPost, c.resultEndpoint, body)
	if err != nil {
		return err
	}
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
