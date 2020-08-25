package worker

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var (
	jobHeader = http.Header{
		"X-Tile":   {"job"},
		"X-Height": {"4242"},
		"X-Width":  {"42"},
	}

	testJob = Job{
		TileName: "job",
		Height:   4242,
		Width:    42,
	}
)

func TestMarshalJobToHeader(t *testing.T) {
	header := http.Header{}
	MarshalJobToHeader(&testJob, header)
	require.Equal(t, jobHeader, header)
}

func TestParseJobFromHTTP(t *testing.T) {
	resp := &http.Response{
		Header: jobHeader,
	}
	result, err := ParseJobFromHTTP(resp)
	require.NoError(t, err)
	require.Equal(t, testJob, result)
}

func TestHTTPClient_Subscribe(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/work/poll", r.URL.Path)
		require.Equal(t, "keep-alive", r.Header.Get("Connection"))

		if requestCount == 0 {
			requestCount++
			w.WriteHeader(http.StatusNotModified)
			return
		}
		MarshalJobToHeader(&testJob, w.Header())
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := HTTPClient{
		client:       server.Client(),
		pollEndpoint: server.URL + "/work/poll",
	}

	ctx, cancelFn := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancelFn()

	err := c.Subscribe(ctx, func(job *Job) error {
		expected := testJob
		expected.Src = http.NoBody
		require.Equal(t, &expected, job)
		return nil
	})

	// got more than 1 requestCount
	require.Equal(t, 1, requestCount)
	require.Equal(t, ErrCancelled, err)
}

func TestHTTPClient_SendResult(t *testing.T) {
	body := "i'm a video"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/work/result", r.URL.Path)

		require.Equal(t, "attachment; filename=8k_video", r.Header.Get("Content-Disposition"))
		require.Equal(t, "application/octet-stream", r.Header.Get("Content-Type"))

		b, err := ioutil.ReadAll(r.Body)
		require.NoError(t, err)

		require.Equal(t, "i'm a video", string(b))
	}))
	defer server.Close()

	c := HTTPClient{
		client:         server.Client(),
		resultEndpoint: server.URL + "/work/result",
	}
	err := c.SendResult("8k_video", strings.NewReader(body))
	require.NoError(t, err)
}
