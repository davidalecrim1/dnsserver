package main

import (
	"context"
	dnsserver "dnsserver"
	"log"
	"log/slog"
	"net"
	"os/signal"
	"syscall"
)

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	slog.Info("Starting DNS server...")

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	conn, err := net.ListenPacket("udp", ":2053")
	if err != nil {
		log.Fatal(err)
	}

	s := dnsserver.NewServer()
	s.ListenAndServe(ctx, conn)
}
