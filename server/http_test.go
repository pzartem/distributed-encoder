package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"distributed-encoder/worker"
)

func TestHTTPHandler_AcceptResult(t *testing.T) {
	body := strings.NewReader("")
	fileName := "file.mp4"
	req, err := http.NewRequest(http.MethodPost, "/result", body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Disposition", "attachment; filename="+fileName)

	var serviceMock serverMock
	h := HTTPHandler{Service: &serviceMock}

	serviceMock.On("AcceptResult", fileName, body).Once()

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(h.AcceptResult)
	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
}

func TestHTTPHandler_Dispatch(t *testing.T) {
	body := strings.NewReader("")
	req, err := http.NewRequest(http.MethodPost, "/jobs", body)
	if err != nil {
		t.Fatal(err)
	}
	var serviceMock serverMock
	h := HTTPHandler{Service: &serviceMock}

	serviceMock.On("Dispatch").
		Return(nil, ErrDispatchTimeout).
		Once()

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(h.Dispatch)
	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusNotModified, rr.Code)
}

type serverMock struct {
	mock.Mock
}

func (s *serverMock) Dispatch() (*worker.Job, error) {
	args := s.Mock.Called()
	err := args.Get(1).(error)

	return nil, err
}

func (s *serverMock) AcceptResult(s2 string, reader io.Reader) error {
	return nil
}

func (s *serverMock) TriggerWork(request EncodeVideoRequest) error {
	panic("implement me")
}
