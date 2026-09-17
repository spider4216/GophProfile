package middlewares

import (
	"net/http"
	"time"
)

func (m Middleware) WithLogging(h http.Handler) http.Handler {
	logFn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		url := r.RequestURI
		method := r.Method

		respData := &responseData{
			status: 0,
			size:   0,
		}

		lw := loggingResponseWriter{
			ResponseWriter: w,
			responseData:   respData,
		}

		h.ServeHTTP(&lw, r)

		duration := time.Since(start)

		m.logger.Info("Info",
			"url", url,
			"method", method,
			"duration", duration,
			"status", respData.status,
			"size", respData.size,
		)
	}

	return http.HandlerFunc(logFn)
}

type responseData struct {
	status int
	size   int
}

type loggingResponseWriter struct {
	http.ResponseWriter
	responseData *responseData
}

func (r loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size = size

	return size, err
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode
}
