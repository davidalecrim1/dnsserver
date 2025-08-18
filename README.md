# DNS Server

A lightweight, high-performance DNS server implementation in Go that supports both local query processing and query forwarding to upstream resolvers.

This was implemented from scratch to learn low level details from UDP/DNS protocol.

## Features

- **Pure Go Implementation**: Built entirely in Go for cross-platform compatibility and easy deployment
- **Dual Mode Operation**: 
  - Local query processing with custom response logic
  - Forwarding mode for delegating queries to upstream DNS resolvers
- **UDP Protocol Support**: Optimized for UDP-based DNS queries
- **Graceful Shutdown**: Proper signal handling for clean server termination
- **Configurable Resolver**: Easy configuration of upstream DNS resolvers
- **Comprehensive DNS Protocol Support**: Implements DNS header, questions, and answers according to RFC standards

### Core Components

- **Header**: DNS message header with proper bit manipulation for flags
- **Question**: DNS query structure with name parsing
- **Answer**: DNS response record with TTL and data
- **Message**: Complete DNS message container
- **Server**: Main server instance with configurable forwarding

## Running the Server

### Local Mode (Default)
```bash
go run cmd/server/main.go
```

### Forwarding Mode
```bash
go run cmd/server/main.go --resolver=1.1.1.1:53
```

The server will start listening on UDP port 2053.

### Testing

Use the provided Makefile targets for quick testing:

```bash
# Test with netcat
make simple-test-1

# Test with dig
make simple-test-2

# Run with hot reload (requires air)
make run

# Run with forwarding enabled
make run-with-forwarding
```