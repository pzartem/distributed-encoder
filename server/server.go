package server

import (
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"path"
	"time"

	"distributed-encoder/transcoder"
	"distributed-encoder/worker"
)

var (
	// ErrDispatchTimeout is returned when a Dispatch function ends
	ErrDispatchTimeout = errors.New("dispatch timeout")
)

// EncodeVideoRequest represents parameters of the video encode request
type EncodeVideoRequest struct {
	// Tiles is an amount of tiles for the video
	Tiles int

	// Height is a resolution of the video
	Height int

	// Width is a resolution of the video
	Width int

	// FilePath is a path for the file
	FilePath string
}

// Store is a store for the service
type Store interface {
	WriteObject(key string, src io.Reader) error
	HasObject(key string) bool
}

// TileStreamer is a real-time stream of the tile
type TileStreamer interface {
	StreamTile(args *transcoder.CropArgs) (io.ReadCloser, error)
}

type tileJob struct {
	TileNum int
	File    string
	Path    string

	PosX int
	PosY int

	Width  int
	Height int
}

// Config represents available server configuration
type Config struct {
	// DispatchTimeout is a maximum wait time for the client per job request session 15 seconds is a default
	DispatchTimeout time.Duration

	// Store is a store for the results
	Store Store
	// TileStreamer is a video tile stream
	TileStreamer TileStreamer
}

// Server splits a video file into tile jobs and distributes it as a byte stream to clients
type Server struct {
	store        Store
	tileStreamer TileStreamer

	dispatchTimeout time.Duration
	jobChan         chan tileJob
}

// New creates a new server
func New(cfg Config) (*Server, error) {
	if cfg.TileStreamer == nil {
		return nil, fmt.Errorf("tilestreamer is empty")
	}
	if cfg.Store == nil {
		return nil, fmt.Errorf("store is empty")
	}
	if cfg.DispatchTimeout != time.Duration(0) {
		cfg.DispatchTimeout = 15 * time.Second
	}

	s := &Server{
		store:           cfg.Store,
		tileStreamer:    cfg.TileStreamer,
		dispatchTimeout: cfg.DispatchTimeout,

		jobChan: make(chan tileJob),
	}

	return s, nil
}

// TriggerWork triggers video encoding work
func (s *Server) TriggerWork(request EncodeVideoRequest) error {
	log.Printf("Work is triggered %+v", request)
	if !s.store.HasObject(request.FilePath) {
		return fmt.Errorf("file: %s is not found in a storage", request.FilePath)
	}
	go func() {
		buildCropJobs(request, func(job tileJob) {
			log.Printf("[Job] enqueued: %s-%v", job.Path, job.TileNum)
			s.jobChan <- job
		})
	}()

	return nil
}

// Dispatch sends a tile job stream when jobs are requested
// When timeout is reached returns ErrDispatchTimeout error
func (s *Server) Dispatch() (*worker.Job, error) {
	select {
	case job := <-s.jobChan:
		log.Printf("Dispatching job: %s, tile: %v", job.Path, job.TileNum)
		stream, err := s.tileStreamer.StreamTile(&transcoder.CropArgs{
			Input:  job.Path,
			X:      job.PosX,
			Y:      job.PosY,
			Height: job.Height,
			Width:  job.Width,
		})
		if err != nil {
			return nil, err
		}
		return &worker.Job{
			TileName: fmt.Sprint(job.File, "_", job.TileNum),
			Width:    job.Width,
			Height:   job.Height,
			Src:      stream,
		}, nil
	case <-time.After(s.dispatchTimeout):
		return nil, ErrDispatchTimeout
	}
}

// AcceptResult receives the result stream and saves it to the store
func (s *Server) AcceptResult(name string, input io.Reader) error {
	if err := s.store.WriteObject(name, input); err != nil {
		return err
	}
	return nil
}

func (s *Server) Close() error {
	close(s.jobChan)
	return nil
}

func buildCropJobs(req EncodeVideoRequest, jobFunc func(tileJob)) {
	_, file := path.Split(req.FilePath)

	cols, rows := calcColumnRows(req.Tiles)

	wRes := req.Width / cols
	hRes := req.Height / rows

	tileNum := 0
	for x := 0; x < req.Width; x += wRes {
		for y := 0; y < req.Height; y += hRes {
			jobFunc(tileJob{
				TileNum: tileNum,
				File:    file,
				Path:    req.FilePath,
				PosX:    x,
				PosY:    y,
				Width:   wRes,
				Height:  hRes,
			})
			tileNum++
		}
	}
}

func calcColumnRows(tiles int) (col int, rows int) {
	numColumns := int(math.Sqrt(float64(tiles)))
	numRows := tiles / numColumns

	return numColumns, numRows
}
