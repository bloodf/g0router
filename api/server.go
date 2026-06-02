package api

import (
	"fmt"
	"net"
	"strconv"

	"github.com/bloodf/g0router/api/handlers"
	"github.com/valyala/fasthttp"
)

type ServerConfig struct {
	Port            int
	Version         string
	RequireAPIKey   bool
	APIKeySecret    string
	APIKeyValidator APIKeyValidator
}

type Server struct {
	config ServerConfig
	server *fasthttp.Server
}

func NewServer(config ServerConfig) *Server {
	srv := &Server{config: config}
	srv.server = &fasthttp.Server{
		Handler: srv.handle,
	}
	return srv
}

func (s *Server) Serve(ln net.Listener) error {
	if err := s.server.Serve(ln); err != nil {
		return fmt.Errorf("serve: %w", err)
	}
	return nil
}

func (s *Server) Stop() error {
	if err := s.server.Shutdown(); err != nil {
		return fmt.Errorf("stop server: %w", err)
	}
	return nil
}

func (s *Server) listener() net.Listener {
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(s.config.Port))
	if err != nil {
		return nil
	}
	return ln
}

func (s *Server) handle(ctx *fasthttp.RequestCtx) {
	if !s.applyMiddleware(ctx) {
		return
	}

	switch string(ctx.Path()) {
	case "/healthz":
		handlers.Health(ctx, s.config.Version)
	default:
		ctx.SetStatusCode(fasthttp.StatusNotFound)
	}
}
