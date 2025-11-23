// securebus/securebus_v2_test.go
package tests

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"io"
	"sync"
	"testing"
	"time"
	"weriKana/internal/bank"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper: in-memory pipe that simulates network
func newConnPipe() (pipe, clientRW, serverRW io.ReadWriter) {
	client := &partialPipe{}
	server := &partialPipe{}
	return pipe{client, server}, client, server
}

type pipe struct {
	a, b *partialPipe
}

type partialPipe struct {
	buf bytes.Buffer
	mu  sync.Mutex
}

func (p *partialPipe) Write(data []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	n, err := p.buf.Write(data)
	if n < len(data) && err == nil {
		err = io.ErrShortWrite
	}
	return n, err
}

func (p *partialPipe) Read(data []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.buf.Read(data)
}

// ----------------------------
// Test: Basic Handshake + Send/Receive
// ----------------------------
func TestBasicHandshakeAndMessaging(t *testing.T) {
	clientKey, _ := GenerateIdentity()
	serverKey, _ := GenerateIdentity()

	_, clientRW, serverRW := newConnPipe()

	// Server
	serverReady := make(chan *SecureConn)
	go func() {
		hs := NewHandshaker(serverKey, &clientKey.Public, false)
		conn, _, _ := hs.Perform(context.Background(), serverRW, 0)
		serverReady <- conn
	}()

	// Client
	clientHS := NewHandshaker(clientKey, &serverKey.Public, true)
	clientConn, _, err := clientHS.Perform(context.Background(), clientRW, 0)
	require.NoError(t, err)

	serverConn := <-serverReady

	// Send message
	require.NoError(t, clientConn.Send(context.Background(), "test.topic", "hello world"))

	var msg string
	topic, err := serverConn.Receive(context.Background(), &msg)
	require.NoError(t, err)
	require.Equal(t, "test.topic", topic)
	require.Equal(t, "hello world", msg)
}

// ----------------------------
// Test: Cross-Session Replay Attack Fails
// ----------------------------
func TestCrossSessionReplayRejected(t *testing.T) {
	clientKey, _ := GenerateIdentity()
	serverKey, _ := GenerateIdentity()

	// First session
	_, rw1a, rw1b := newConnPipe()
	clientHS := NewHandshaker(clientKey, &serverKey.Public, true)
	conn1, _, _ := clientHS.Perform(context.Background(), rw1a, 0)
	serverHS := NewHandshaker(serverKey, &clientKey.Public, false)
	serverConn1, _, _ := serverHS.Perform(context.Background(), rw1b, 0)

	// Capture a legitimate encrypted message
	require.NoError(t, conn1.Send(context.Background(), "test", "secret"))
	typ, encrypted, err := readFrame(context.Background(), serverConn1.rw)
	require.NoError(t, err)
	require.Equal(t, byte(0x00), typ)

	// New session (different keys derived)
	_, rw2a, rw2b := newConnPipe()
	conn2, _, _ := clientHS.Perform(context.Background(), rw2a, 1000) // high counter
	serverConn2, _, _ := serverHS.Perform(context.Background(), rw2b, 1000)

	// Try to replay old encrypted message into new session
	require.NoError(t, writeFrame(context.Background(), serverConn2.rw, 0x00, encrypted))

	var dummy string
	_, err = serverConn2.Receive(context.Background(), &dummy)
	require.Error(t, err)
	require.Contains(t, err.Error(), "different session") // SessionTag mismatch
}

// ----------------------------
// Test: Same Session Replay Rejected (within window)
// ----------------------------
func TestSameSessionReplayRejected(t *testing.T) {
	clientKey, _ := GenerateIdentity()
	serverKey, _ := GenerateIdentity()

	_, clientRW, serverRW := newConnPipe()

	clientHS := NewHandshaker(clientKey, &serverKey.Public, true)
	clientConn, _, _ := clientHS.Perform(context.Background(), clientRW, 0)

	serverHS := NewHandshaker(serverKey, &clientKey.Public, false)
	serverConn, _, _ := serverHS.Perform(context.Background(), serverRW, 0)

	// Send 5 messages
	for i := 0; i < 5; i++ {
		require.NoError(t, clientConn.Send(context.Background(), "seq", i))
	}

	// Receive all
	for i := 0; i < 5; i++ {
		var v int
		_, err := serverConn.Receive(context.Background(), &v)
		require.NoError(t, err)
		require.Equal(t, i, v)
	}

	// Now try to re-receive (replay from network buffer)
	_, err := serverConn.Receive(context.Background(), nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "replay")
}

// ----------------------------
// Test: Session Resumption with Persistent Counter
// ----------------------------
func TestSessionResumptionWithPersistentCounter(t *testing.T) {
	clientKey, _ := GenerateIdentity()
	serverKey, _ := GenerateIdentity()

	// First full handshake
	_, rw1a, rw1b := newConnPipe()
	clientHS := NewHandshaker(clientKey, &serverKey.Public, true)
	clientConn1, meta1, _ := clientHS.Perform(context.Background(), rw1a, 100) // start at 100
	serverHS := NewHandshaker(serverKey, &clientKey.Public, false)
	_, _, _ = serverHS.Perform(context.Background(), rw1b, 0)

	// Send some messages
	for i := 0; i < 3; i++ {
		require.NoError(t, clientConn1.Send(context.Background(), "resume", i))
	}

	// Resume session with persisted counter
	_, rw2a, _ := newConnPipe()
	clientConn2, err := clientHS.Resume(context.Background(), rw2a, meta1)
	require.NoError(t, err)

	// Next counter should continue from last used
	require.NoError(t, clientConn2.Send(context.Background(), "after-resume", 999))

	// Counter should be 104 now (100 + 3 + 1)
	require.Equal(t, uint64(103), clientConn2.counter) // last sent was 103 → next is 104
}

// ----------------------------
// Test: Oversized Frame Still Rejected
// ----------------------------
func TestOversizedFrameRejected(t *testing.T) {
	data := make([]byte, MaxFrameSize)
	err := writeFrame(context.Background(), &partialPipe{}, 0x00, data)
	require.Error(t, err)
	require.Contains(t, err.Error(), "frame size invalid")
}

// ----------------------------
// Test: Context Cancellation During I/O
// ----------------------------
func TestContextCancellation(t *testing.T) {
	clientKey, _ := GenerateIdentity()
	_, clientRW, _ := newConnPipe()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	clientHS := NewHandshaker(clientKey, nil, true)
	_, _, err := clientHS.Perform(ctx, clientRW, 0)
	require.Error(t, err)
	require.True(t, errors.Is(err, context.Canceled))
}

// ----------------------------
// Test: Complex Payload Round-Trip
// ----------------------------
func TestComplexPayload(t *testing.T) {
	type Payload struct {
		ID     int
		Name   string
		Active bool
		Tags   []string
		Meta   map[string]any
	}

	clientKey, _ := GenerateIdentity()
	serverKey, _ := GenerateIdentity()

	_, clientRW, serverRW := newConnPipe()

	go func() {
		hs := NewHandshaker(serverKey, &clientKey.Public, false)
		conn, _, _ := hs.Perform(context.Background(), serverRW, 0)
		var p Payload
		conn.Receive(context.Background(), &p)
	}()

	clientHS := NewHandshaker(clientKey, &serverKey.Public, true)
	clientConn, _, _ := clientHS.Perform(context.Background(), clientRW, 0)

	p := Payload{
		ID:     42,
		Name:   "test-device",
		Active: true,
		Tags:   []string{"sensor", "v2"},
		Meta:   map[string]any{"version": 2.0},
	}

	require.NoError(t, clientConn.Send(context.Background(), "device.update", p))
}

// ----------------------------
// Benchmark: High-Throughput Messaging
// ----------------------------
func BenchmarkSecureBus(b *testing.B) {
	clientKey, _ := GenerateIdentity()
	serverKey, _ := GenerateIdentity()

	_, clientRW, serverRW := newConnPipe()

	go func() {
		hs := NewHandshaker(serverKey, &clientKey.Public, false)
		conn, _, _ := hs.Perform(context.Background(), serverRW, 0)
		var x int
		for {
			conn.Receive(context.Background(), &x)
		}
	}()

	clientHS := NewHandshaker(clientKey, &serverKey.Public, true)
	clientConn, _, _ := clientHS.Perform(context.Background(), clientRW, 0)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			clientConn.Send(context.Background(), "bench", i)
			i++
		}
	})
}
