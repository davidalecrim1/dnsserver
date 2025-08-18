package dnsserver

import (
	"context"
	"log/slog"
	"net"
	"time"
)

type Options struct {
	Resolver string
}

type Server struct {
	opts Options
}

func NewServer(opts Options) *Server {
	return &Server{opts: opts}
}

func (s *Server) shouldForwardQuery() bool {
	return s.opts.Resolver != ""
}

func (s *Server) ListenAndServe(ctx context.Context, conn net.PacketConn) {
	defer conn.Close()

	if s.shouldForwardQuery() {
		slog.Info("Forwarding requests to resolver", "resolver", s.opts.Resolver)
	}

	buf := make([]byte, 1024)
	for {
		select {
		case <-ctx.Done():
			slog.Info("Received interrupt signal, shutting down...")
			return
		default:
			conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
			n, addr, err := conn.ReadFrom(buf)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				slog.Error("Error reading from connection", "error", err)
				return
			}

			slog.Debug("Received request", "n", n, "addr", addr, "buf", buf[:n])

			if s.shouldForwardQuery() {
				s.handleForwardedQuery(conn, addr, buf[:n])
			} else {
				s.handleLocalQuery(conn, addr, buf[:n])
			}
		}
	}
}

func (s *Server) handleLocalQuery(conn net.PacketConn, addr net.Addr, queryBytes []byte) {
	msg, err := NewMessageFromBytes(queryBytes)
	if err != nil {
		slog.Error("Error parsing message", "error", err)
		return
	}

	msg.ProcessQuestions()
	msgBytes, err := msg.MarshalBinary()
	if err != nil {
		slog.Error("Error marshalling message", "error", err)
		return
	}

	slog.Debug("Sending response", "msg", msg, "msgBytes", msgBytes)
	conn.WriteTo(msgBytes, addr)
}

func (s *Server) handleForwardedQuery(conn net.PacketConn, addr net.Addr, queryBytes []byte) {
	responseBytes, err := s.forwardQuery(queryBytes)
	if err != nil {
		slog.Error("Error forwarding query, continuing with local processing", "error", err, "resolver", s.opts.Resolver)
		s.handleForwardingError(conn, addr, queryBytes)
		return
	}
	conn.WriteTo(responseBytes, addr)
}

func (s *Server) handleForwardingError(conn net.PacketConn, addr net.Addr, queryBytes []byte) {
	msg, err := NewMessageFromBytes(queryBytes)
	if err != nil {
		slog.Error("Error parsing message", "error", err)
		return
	}

	msg.Header.SetResponseCode(RCODE_SERVER_FAILURE)
	msg.Header.AdditionalCount = 0
	raw, err := msg.MarshalBinary()
	if err != nil {
		slog.Error("Error marshalling message", "error", err)
		return
	}

	conn.WriteTo(raw, addr)
}

func (s *Server) forwardQuery(queryBytes []byte) ([]byte, error) {
	conn, err := net.Dial("udp", s.opts.Resolver)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(100 * time.Millisecond))
	_, err = conn.Write(queryBytes)
	if err != nil {
		return nil, err
	}

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}

	return buf[:n], nil
}
