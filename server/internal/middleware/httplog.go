package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/MrPunder/sirius-loyality-system/internal/logger"
)

// httpLogger is logger interface for middleware logger
type httpLogger interface {
	logger.Logger
	RequestLog(method string, path string)
	ResponseLog(status int, size int, duration time.Duration)
}

type responseData struct {
	status int
	size   int
}

type HTTPLogger struct {
	log httpLogger
}

func NewHTTPLoger(logger httpLogger) *HTTPLogger {
	return &HTTPLogger{logger}
}

// loggingResponseWriter allows use ResponnseWriter and stores information to log
type loggingResponseWriter struct {
	http.ResponseWriter
	responseData *responseData
}

func (r *loggingResponseWriter) Write(b []byte) (int, error) {

	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size
	if r.responseData.status == 0 {
		r.responseData.status = 200
	}
	return size, err
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {

	r.responseData.status = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (l *HTTPLogger) HTTPLogHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers := r.Header

		method := r.Method
		path := r.RequestURI
		l.log.RequestLog(method, path)
		l.log.Info(fmt.Sprintf("Headers:  %v", headers))

		start := time.Now()

		resD := &responseData{}

		lw := &loggingResponseWriter{
			ResponseWriter: w,
			responseData:   resD,
		}

		next.ServeHTTP(lw, r)

		l.log.Info("request served from HttpLogger")
		duration := time.Since(start)

		l.log.ResponseLog(resD.status, resD.size, duration)

	})
}
