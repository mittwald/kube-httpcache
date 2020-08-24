package readiness

import (
	"net"
	"net/http"
)

type Server struct {
	addr  string
	ready bool
}

func NewServer(addr string, ready chan struct{}) *Server {
	s := Server{addr, false}

	go func() {
		_, _ = <-ready
		s.ready = true
	}()

	return &s
}

func (s *Server) Run() error {
	l, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}

	server := http.Server{
		Addr:    s.addr,
		Handler: s,
	}

	err = server.Serve(l)
	if err == http.ErrServerClosed {
		return nil
	}

	return err
}

func (s *Server) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if s.ready {
		writer.WriteHeader(http.StatusNoContent)
		return
	}

	writer.WriteHeader(http.StatusServiceUnavailable)
}
