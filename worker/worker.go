package worker

import (
	"io"
	"log"
)

type Job struct {
	TileName string
	Height   int
	Width    int
	Src      io.ReadCloser
}

type HandleJobFunc func(Job) error

// Client is the consumer client for a worker
type Client interface {
	Subscribe(HandleJobFunc) error
	SendResult(filename string, src io.Reader) error
}

// VideoEncoder encodes video as a stream
type VideoEncoder interface {
	Encode(reader io.Reader) (io.ReadCloser, error)
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
func (w *Worker) Start() error {
	err := w.client.Subscribe(func(job Job) error {
		log.Println("Job received: ")
		err := w.work(job)
		if err != nil {
			log.Println(err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (w *Worker) work(job Job) error {
	output, err := w.encoder.Encode(job.Src)
	if err != nil {
		return err
	}
	defer output.Close()

	err = w.client.SendResult(job.TileName, output)
	if err != nil {
		return err
	}

	return nil
}

//	cmd := transcoder.EncodeVideo(transcoder.EncodeOps{
//	Height: job.Height,
//	Width:  job.Width,
//})
//cmd.Stdin = job.Src
//out, err := cmd.StdoutPipe()
//if err != nil {
//	return err
//}
//defer out.Close()
//if err := cmd.Start(); err != nil {
//	return err
//}
//
//go func() {
//	err = w.streamResult(fmt.Sprintf("%s.ts", job.TileName), out)
//	if err != nil {
//		println(err)
//	}
//	fmt.Println("done")
//}()
//
//if err := cmd.Wait(); err != nil {
//	return err
//}
//
//return nil
