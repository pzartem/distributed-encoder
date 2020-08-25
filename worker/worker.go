package worker

import (
	"bufio"
	"context"
	"io"
	"log"

	"distributed-encoder/transcoder"
)

// Client is the consumer client for a worker
type Client interface {
	Subscribe(context.Context, HandleJobFunc) error
	SendResult(filename string, src io.Reader) error
}

// VideoEncoder encodes video as a stream
type VideoEncoder interface {
	Encode(reader io.Reader, args transcoder.EncodeArgs) (io.ReadCloser, error)
}

// Worker accepts jobs from the server process them and returns the result
type Worker struct {
	client  Client
	encoder VideoEncoder
}

// New creates a new worker
func New(client Client, encoder VideoEncoder) (*Worker, error) {
	w := Worker{
		client:  client,
		encoder: encoder,
	}

	return &w, nil
}

// Start starts worker and blo
func (w *Worker) Start(ctx context.Context) error {
	err := w.client.Subscribe(ctx, func(job *Job) error {
		log.Printf("Job received: %+v", job)
		err := w.work(job)
		if err != nil {
			log.Println("Error work:", err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (w *Worker) work(job *Job) error {
	output, err := w.encoder.Encode(job.Src, transcoder.EncodeArgs{
		Height: job.Height,
		Width:  job.Width,
	})
	if err != nil {
		return err
	}
	defer output.Close()
	err = w.client.SendResult(job.TileName, bufio.NewReader(output))
	if err != nil {
		return err
	}
	return nil
}

// Job represents worker's job
type Job struct {
	TileName string
	Height   int
	Width    int
	Src      io.ReadCloser
}
