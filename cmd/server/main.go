package main

import (
	"context"
	dnsserver "dnsserver"
	"flag"
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

	resolver := flag.String("resolver", "", "The resolver to forward requests to")
	flag.Parse()

	opts := dnsserver.Options{
		Resolver: *resolver,
	}

	s := dnsserver.NewServer(opts)
	s.ListenAndServe(ctx, conn)
}
