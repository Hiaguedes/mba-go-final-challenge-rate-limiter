package webserver

import (
	"log"
	"net/http"

	"github.com/hiaguedes/mba-go-final-challenge-rate-limiter/internal/controller"
)


type WebServer struct {
	Router        *http.ServeMux
	Handlers      map[string]http.HandlerFunc
	WebServerPort string
}

func NewWebServer(serverPort string) *WebServer {
	return &WebServer{
		Router:        http.NewServeMux(),
		Handlers:      make(map[string]http.HandlerFunc),
		WebServerPort: serverPort,
	}
}

func (s *WebServer) AddHandler(path string, handler http.HandlerFunc) {
	s.Handlers[path] = handler
}

func (s *WebServer) Handler() http.Handler {
	rateLimitController := controller.NewRateLimitMiddleware()
	for path, handler := range s.Handlers {
		s.Router.Handle(path, rateLimitController.RateLimitMiddleware(handler))
	}
	return s.Router
}

func (s *WebServer) Start() error {
	logger := log.Default()
	logger.Println("Server iniciado na porta ", s.WebServerPort)

	return http.ListenAndServe(s.WebServerPort, s.Handler())
}