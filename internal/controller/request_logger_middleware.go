package controller

import (
	"log"
	"net"
	"net/http"
	"time"
)

// statusRecorder guarda o status code enviado pelo handler interno.
type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (s *statusRecorder) WriteHeader(code int) {
	if !s.wroteHeader {
		s.status = code
		s.wroteHeader = true
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if !s.wroteHeader {
		s.WriteHeader(http.StatusOK)
	}
	return s.ResponseWriter.Write(b)
}

// RequestLogger registra uma linha por requisição: método, caminho, status,
// duração, IP do cliente e se o header API_KEY foi enviado (sem o valor do token).
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rec, r)

		clientHost, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			clientHost = r.RemoteAddr
		}

		log.Printf(
			"method=%s path=%s status=%d duration=%s client=%s api_key_sent=%t",
			r.Method,
			r.URL.Path,
			rec.status,
			time.Since(start).Round(time.Microsecond),
			clientHost,
			r.Header.Get(apiKeyHeader) != "",
		)
	})
}
