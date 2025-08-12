package dnsserver

import (
	"context"
	"log/slog"
	"net"
	"time"
)

type Server struct{}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) ListenAndServe(ctx context.Context, conn net.PacketConn) {
	defer conn.Close()

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
			slog.Debug("Received request", "n", n, "addr", addr)

			msg, err := NewMessageFromBytes(buf[:n])
			if err != nil {
				slog.Error("Error parsing message", "error", err)
				continue
			}

			slog.Debug("Sending response", "msg", msg)
			msgBytes, err := msg.MarshalBinary()
			if err != nil {
				slog.Error("Error marshalling message", "error", err)
				continue
			}

			conn.WriteTo(msgBytes, addr)
		}
	}
}
