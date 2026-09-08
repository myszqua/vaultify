package tripper

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/myszqua/vaultify/vault/logger"
)

const (
	maxBodyLogBytes     = 500
	maxRespLogBytes     = 1000
	errorStatusBoundary = 400
)

// LoggingRoundTripper wraps an http.RoundTripper and logs every HTTP request
// and response through the given logger. Bodies are logged when logBody is true.
type LoggingRoundTripper struct {
	proxy   http.RoundTripper
	logger  logger.Writer
	logBody bool
}

// NewLoggingRoundTripper builds a LoggingRoundTripper around proxy, falling
// back to http.DefaultTransport when proxy is nil.
func NewLoggingRoundTripper(proxy http.RoundTripper, logBody bool, logger logger.Writer) *LoggingRoundTripper {
	if proxy == nil {
		proxy = http.DefaultTransport
	}

	return &LoggingRoundTripper{
		proxy:   proxy,
		logBody: logBody,
		logger:  logger,
	}
}

// RoundTrip executes req and logs the request and its response.
func (l *LoggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()

	l.logRequest(req)

	resp, err := l.proxy.RoundTrip(req)
	duration := time.Since(start)

	if err != nil {
		l.logger.Warn("request failed", "error", err, "duration_ms", duration.Milliseconds())

		return resp, err
	}

	l.logResponse(resp, duration)

	return resp, nil
}

func (l *LoggingRoundTripper) logRequest(req *http.Request) {
	l.logger.Info("request",
		"method", req.Method,
		"url", req.URL.String(),
	)

	if l.logBody && req.Body != nil {
		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			req.Body.Close()

			return
		}

		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))

		l.logBodyBytes("request body", bodyBytes, maxBodyLogBytes)
	}
}

func (l *LoggingRoundTripper) logBodyBytes(prefix string, bodyBytes []byte, maxBytes int) {
	if len(bodyBytes) == 0 {
		return
	}

	bodyStr := string(bodyBytes)
	if len(bodyStr) > maxBytes {
		bodyStr = bodyStr[:maxBytes] + "... (truncated)"
	}

	l.logger.Debug(prefix, "body", bodyStr)
}

func (l *LoggingRoundTripper) logResponse(resp *http.Response, duration time.Duration) {
	status := resp.StatusCode

	if status >= errorStatusBoundary {
		l.logger.Warn("response error",
			"status", status,
			"duration_ms", duration.Milliseconds(),
		)
	} else {
		l.logger.Info("response",
			"status", status,
			"duration_ms", duration.Milliseconds(),
		)
	}

	if l.logBody && resp.Body != nil {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			resp.Body.Close()

			return
		}

		resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))

		l.logBodyBytes("response body", bodyBytes, maxRespLogBytes)
	}
}