package throtto

import (
	"fmt"
	"net/http"
)

func limitHandler(l *limiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l.lcount.Lock()
		l.lcount.count++
		l.lcount.Unlock()
		if !l.allow() {
			l.add(exceed)
			if l.rejectFunc != nil {
				l.rejectFunc(w, r)
			}
			http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			return
		}
		rw := newCustomResponseWriter(w)
		next.ServeHTTP(rw, r)
		if err := l.next(rw.StatusCode); err != nil {
			l.debugln(fmt.Sprintf("err=%s", err.Error()))
		}
	})
}

type customResponseWriter struct {
	http.ResponseWriter
	StatusCode int
}

func newCustomResponseWriter(w http.ResponseWriter) *customResponseWriter {
	return &customResponseWriter{w, http.StatusOK}
}

func (crw *customResponseWriter) WriteHeader(statusCode int) {
	crw.StatusCode = statusCode
	crw.ResponseWriter.WriteHeader(statusCode)
}

func getStatus(code int) string {
	if code >= http.StatusInternalServerError {
		return failure
	}
	return success
}
