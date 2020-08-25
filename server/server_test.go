package server

import (
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"distributed-encoder/transcoder"
	"distributed-encoder/worker"
)

func TestNew(t *testing.T) {
	store := &FSObjectStore{}
	encoder := &transcoder.Transcoder{}

	tests := map[string]struct {
		cfg     Config
		want    *Server
		wantErr bool
	}{
		"correct usage": {
			cfg: Config{
				DispatchTimeout: 10 * time.Second,
				Store:           store,
				TileStreamer:    encoder,
			},
			want: &Server{
				store:           store,
				tileStreamer:    encoder,
				dispatchTimeout: 10 * time.Second,
			},
		},
		"default timeout": {
			cfg: Config{
				Store:        store,
				TileStreamer: encoder,
			},
			want: &Server{
				store:           store,
				tileStreamer:    encoder,
				dispatchTimeout: 15 * time.Second,
			},
		},
		"no store": {
			cfg: Config{
				TileStreamer: encoder,
			},
			wantErr: true,
		},
		"no encoder": {
			cfg: Config{
				Store: store,
			},
			wantErr: true,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := New(tt.cfg)
			if err != nil && tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want.store, got.store)
			require.Equal(t, tt.want.tileStreamer, got.tileStreamer)
			require.Equal(t, tt.want.dispatchTimeout, got.dispatchTimeout)
			require.NotNil(t, got.jobChan)
		})
	}
}

func TestServer_AcceptResult(t *testing.T) {
	var mock storeMock
	s := Server{
		store: &mock,
	}

	reader := strings.NewReader("file")
	mock.On("WriteObject", "input.mp4", reader).Once()
	err := s.AcceptResult("input.mp4", strings.NewReader("file"))
	require.NoError(t, err)
}

type storeMock struct {
	mock.Mock
}

func (s *storeMock) WriteObject(key string, src io.Reader) error {
	return nil
}

func (s *storeMock) HasObject(key string) bool {
	panic("implement me")
}

func TestServer_Dispatch(t *testing.T) {
	var streamer streamerMock
	s := Server{
		dispatchTimeout: 1 * time.Millisecond,
		tileStreamer:    &streamer,
		jobChan:         make(chan tileJob),
	}

	_, err := s.Dispatch()
	require.Equal(t, ErrDispatchTimeout, err)

	s.dispatchTimeout = 5 * time.Second
	go func() {
		s.jobChan <- tileJob{
			TileNum: 0,
			File:    "file",
			Path:    "path",
			PosX:    1,
			PosY:    2,
			Width:   3,
			Height:  4,
		}
	}()
	streamer.On("StreamTile", &transcoder.CropArgs{
		Input:  "path",
		X:      1,
		Y:      2,
		Width:  3,
		Height: 4,
	}).Once()
	job, err := s.Dispatch()

	require.NoError(t, err)
	require.Equal(t, &worker.Job{
		TileName: "file_tile_0",
		Height:   4,
		Width:    3,
		Src:      nil,
	}, job)
}

type streamerMock struct {
	mock.Mock
}

func (e *streamerMock) StreamTile(args *transcoder.CropArgs) (io.ReadCloser, error) {
	return nil, nil
}

func Test_buildCropJobs(t *testing.T) {
	tests := map[string]struct {
		req      EncodeVideoRequest
		expected []tileJob
	}{
		"4 tiles": {
			req: EncodeVideoRequest{
				Tiles:  4,
				Height: 1280,
				Width:  720,
			},
			expected: []tileJob{
				{
					TileNum: 0,
					PosX:    0,
					PosY:    0,
					Width:   360,
					Height:  640,
				},
				{
					TileNum: 1,
					PosX:    0,
					PosY:    640,
					Width:   360,
					Height:  640,
				},
				{
					TileNum: 2,
					PosX:    360,
					PosY:    0,
					Width:   360,
					Height:  640,
				},
				{
					TileNum: 3,
					PosX:    360,
					PosY:    640,
					Width:   360,
					Height:  640,
				},
			},
		},
		"2 tiles": {
			req: EncodeVideoRequest{
				Tiles:    2,
				Height:   1280,
				Width:    720,
				FilePath: "/tmp/v.mp4",
			},
			expected: []tileJob{
				{
					TileNum: 0,
					File:    "v.mp4",
					Path:    "/tmp/v.mp4",
					PosX:    0,
					PosY:    0,
					Width:   720,
					Height:  640,
				},
				{
					TileNum: 1,
					File:    "v.mp4",
					Path:    "/tmp/v.mp4",
					PosX:    0,
					PosY:    640,
					Width:   720,
					Height:  640,
				},
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var result []tileJob
			buildCropJobs(tt.req, func(job tileJob) {
				result = append(result, job)
			})

			require.Equal(t, tt.expected, result)
		})
	}
}
