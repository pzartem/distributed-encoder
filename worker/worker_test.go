package worker

import (
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/require"

	"distributed-encoder/transcoder"
)

func TestNew(t *testing.T) {
	client := &HTTPClient{}
	encoder := &transcoder.Transcoder{}

	tests := map[string]struct {
		client  Client
		encoder VideoEncoder
		want    *Worker
		wantErr bool
	}{
		"correct usage": {
			client:  client,
			encoder: encoder,
			want: &Worker{
				client:  client,
				encoder: encoder,
			},
			wantErr: false,
		},
		"no client": {
			encoder: encoder,
			wantErr: true,
		},
		"no encoder": {
			encoder: encoder,
			wantErr: true,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := New(tt.client, tt.encoder)
			if err != nil && tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestWorker_Start(t *testing.T) {
	input := newStringReader("i'm a file")

	server := fakeServer(t, &Job{
		TileName: "1",
		Height:   100,
		Width:    200,
		Src:      input,
	}, "i'm an encoded file")

	var encoder encoderMock
	w := Worker{
		client: &HTTPClient{
			client:         server.Client(),
			pollEndpoint:   server.URL + "/poll",
			resultEndpoint: server.URL + "/result",
		},
		encoder: &encoder,
	}
	ctx, cancelFn := context.WithCancel(context.Background())
	go func() {
		cancelFn()
	}()
	err := w.Start(ctx)
	require.Error(t, ErrCancelled, err)
}

func fakeServer(t *testing.T, sendJob *Job, expected string) *httptest.Server {
	router := httprouter.New()
	router.HandlerFunc(http.MethodPost, "/poll", func(w http.ResponseWriter, req *http.Request) {
		require.Equal(t, "/poll", req.URL.Path)
		require.Equal(t, "keep-alive", req.Header.Get("Connection"))

		MarshalJobToHeader(sendJob, w.Header())
		_, err := io.Copy(w, sendJob.Src)
		require.NoError(t, err)
	})
	router.HandlerFunc(http.MethodPost, "/result", func(w http.ResponseWriter, req *http.Request) {
		require.Equal(t, "/result", req.URL.Path)

		result, err := ioutil.ReadAll(req.Body)
		require.NoError(t, err)
		t.Log("asserting result: ", string(result))
		require.Equal(t, expected, string(result))
	})

	return httptest.NewServer(router)
}

type encoderMock struct{}

func (e encoderMock) Encode(reader io.Reader, args transcoder.EncodeArgs) (io.ReadCloser, error) {
	encoded := newStringReader("i'm an encoded file")
	return encoded, nil
}

func newStringReader(src string) io.ReadCloser {
	encoded := ioutil.NopCloser(strings.NewReader(src))
	return encoded
}
