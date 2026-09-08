package tripper

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type spyLogger struct {
	infos  int
	warns  int
	debugs int
}

func (s *spyLogger) Info(...any)  { s.infos++ }
func (s *spyLogger) Warn(...any)  { s.warns++ }
func (s *spyLogger) Debug(...any) { s.debugs++ }

func TestNewLoggingRoundTripper(t *testing.T) {
	t.Run("With proxy", func(t *testing.T) {
		proxy := http.DefaultTransport
		l := NewLoggingRoundTripper(proxy, true, &spyLogger{})
		require.NotNil(t, l)
		assert.Equal(t, proxy, l.proxy)
	})

	t.Run("Nil proxy defaults", func(t *testing.T) {
		l := NewLoggingRoundTripper(nil, false, &spyLogger{})
		require.NotNil(t, l)
		assert.Equal(t, http.DefaultTransport, l.proxy)
	})
}

func TestRoundTripSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	logs := &spyLogger{}
	l := NewLoggingRoundTripper(http.DefaultTransport, true, logs)

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		server.URL,
		bytes.NewBufferString(`{"q":1}`),
	)
	require.NoError(t, err)

	resp, err := l.RoundTrip(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, `{"ok":true}`, string(body))

	assert.Equal(t, 2, logs.infos)
	assert.Equal(t, 0, logs.warns)
	assert.Equal(t, 2, logs.debugs)
}

func TestRoundTripProxyError(t *testing.T) {
	failProxy := roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return nil, errors.New("boom")
	})

	logs := &spyLogger{}
	l := NewLoggingRoundTripper(failProxy, true, logs)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.com", nil)
	require.NoError(t, err)

	resp, err := l.RoundTrip(req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, 1, logs.warns)
}

func TestRoundTripResponseErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`error`))
	}))
	defer server.Close()

	logs := &spyLogger{}
	l := NewLoggingRoundTripper(http.DefaultTransport, false, logs)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	require.NoError(t, err)

	resp, err := l.RoundTrip(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	assert.Equal(t, 1, logs.warns)
	assert.Equal(t, 1, logs.infos)
	assert.Equal(t, 0, logs.debugs)
}

func TestLoggingRequestBodyTruncation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	defer server.Close()

	logs := &spyLogger{}
	l := NewLoggingRoundTripper(http.DefaultTransport, true, logs)

	longBody := bytes.Repeat([]byte("a"), 600)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, server.URL, bytes.NewReader(longBody))
	require.NoError(t, err)

	_, err = l.RoundTrip(req)
	require.NoError(t, err)

	require.Equal(t, 1, logs.debugs)
}

func TestLoggingResponseBodyTruncation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(bytes.Repeat([]byte("b"), 1200))
	}))
	defer server.Close()

	logs := &spyLogger{}
	l := NewLoggingRoundTripper(http.DefaultTransport, true, logs)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	require.NoError(t, err)

	_, err = l.RoundTrip(req)
	require.NoError(t, err)

	require.Equal(t, 1, logs.debugs)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
