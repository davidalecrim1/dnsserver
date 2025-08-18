package dnsserver

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServer(t *testing.T) {
	opts := Options{Resolver: "8.8.8.8:53"}
	server := NewServer(opts)

	require.Equal(t, "8.8.8.8:53", server.opts.Resolver)
}

func TestShouldForwardQuery(t *testing.T) {
	tests := []struct {
		name     string
		resolver string
		expected bool
	}{
		{"with resolver", "8.8.8.8:53", true},
		{"without resolver", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := &Server{opts: Options{Resolver: tt.resolver}}
			result := server.shouldForwardQuery()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHandleLocalQuery(t *testing.T) {
	server := &Server{opts: Options{}}

	conn := &mockPacketConn{}
	addr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345}

	queryBytes := createTestQuery()

	server.handleLocalQuery(conn, addr, queryBytes)

	assert.NotEmpty(t, conn.writtenData)
	require.NotEmpty(t, conn.writtenAddr)
	assert.Equal(t, addr, conn.writtenAddr[0])
}

func TestHandleLocalQueryWithInvalidMessage(t *testing.T) {
	server := &Server{opts: Options{}}

	conn := &mockPacketConn{}
	addr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345}

	invalidQuery := []byte{0, 0, 0, 0}

	server.handleLocalQuery(conn, addr, invalidQuery)

	assert.Empty(t, conn.writtenData)
}

func TestHandleForwardedQuery(t *testing.T) {
	server := &Server{opts: Options{Resolver: "127.0.0.1:53535"}}

	conn := &mockPacketConn{}
	addr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345}

	queryBytes := createTestQuery()

	server.handleForwardedQuery(conn, addr, queryBytes)

	assert.NotEmpty(t, conn.writtenData)
}

func TestHandleForwardingError(t *testing.T) {
	server := &Server{opts: Options{Resolver: "invalid-resolver:53"}}

	conn := &mockPacketConn{}
	addr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345}

	queryBytes := createTestQuery()

	server.handleForwardingError(conn, addr, queryBytes)

	assert.NotEmpty(t, conn.writtenData)
	require.NotEmpty(t, conn.writtenAddr)
	assert.Equal(t, addr, conn.writtenAddr[0])
}

func TestForwardQuery(t *testing.T) {
	server := &Server{opts: Options{Resolver: "127.0.0.1:53535"}}

	queryBytes := createTestQuery()

	_, err := server.forwardQuery(queryBytes)

	assert.Error(t, err)
}

func TestListenAndServeWithContextCancellation(t *testing.T) {
	server := &Server{opts: Options{}}

	conn := &mockPacketConn{}
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	server.ListenAndServe(ctx, conn)

	assert.True(t, conn.closed)
}

func TestListenAndServeLocalMode(t *testing.T) {
	server := &Server{opts: Options{}}

	conn := &mockPacketConn{
		readData: [][]byte{createTestQuery()},
		readAddr: []net.Addr{&net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345}},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	server.ListenAndServe(ctx, conn)

	assert.NotEmpty(t, conn.writtenData)
}

func TestListenAndServeForwardingMode(t *testing.T) {
	server := &Server{opts: Options{Resolver: "127.0.0.1:53535"}}

	conn := &mockPacketConn{
		readData: [][]byte{createTestQuery()},
		readAddr: []net.Addr{&net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345}},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	server.ListenAndServe(ctx, conn)

	assert.NotEmpty(t, conn.writtenData)
}

func TestListenAndServeWithReadTimeout(t *testing.T) {
	server := &Server{opts: Options{}}

	conn := &mockPacketConn{
		readTimeout: true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	go server.ListenAndServe(ctx, conn)

	time.Sleep(100 * time.Millisecond)

	assert.False(t, conn.closed)
}

func TestListenAndServeWithReadError(t *testing.T) {
	server := &Server{opts: Options{}}

	conn := &mockPacketConn{
		readError: true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	server.ListenAndServe(ctx, conn)

	assert.True(t, conn.closed)
}

func createTestQuery() []byte {
	header := NewHeader(12345, 0, 1, 0, 0, 0)
	question := Question{Name: "example.com", Type: 1, Class: 1}

	msg := Message{
		Header:    header,
		Questions: []Question{question},
	}

	msgBytes, _ := msg.MarshalBinary()
	return msgBytes
}

type timeoutError struct{}

func (t *timeoutError) Error() string   { return "timeout" }
func (t *timeoutError) Timeout() bool   { return true }
func (t *timeoutError) Temporary() bool { return false }

type mockPacketConn struct {
	writtenData [][]byte
	writtenAddr []net.Addr
	readData    [][]byte
	readAddr    []net.Addr
	readIndex   int
	readTimeout bool
	readError   bool
	closed      bool
}

func (m *mockPacketConn) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	if m.readError {
		return 0, nil, &net.OpError{Op: "read", Net: "udp", Err: &net.AddrError{}}
	}

	if m.readTimeout {
		return 0, nil, &timeoutError{}
	}

	if m.readIndex >= len(m.readData) {
		return 0, nil, &timeoutError{}
	}

	data := m.readData[m.readIndex]
	addr = m.readAddr[m.readIndex]
	m.readIndex++

	copy(p, data)
	return len(data), addr, nil
}

func (m *mockPacketConn) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	m.writtenData = append(m.writtenData, append([]byte{}, p...))
	m.writtenAddr = append(m.writtenAddr, addr)
	return len(p), nil
}

func (m *mockPacketConn) Close() error {
	m.closed = true
	return nil
}

func (m *mockPacketConn) LocalAddr() net.Addr {
	return &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 2053}
}

func (m *mockPacketConn) SetDeadline(t time.Time) error {
	return nil
}

func (m *mockPacketConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (m *mockPacketConn) SetWriteDeadline(t time.Time) error {
	return nil
}
