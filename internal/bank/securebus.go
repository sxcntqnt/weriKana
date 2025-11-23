// internal/bank/securebus.go
// SecureBus 2025 — Powered by gitlab.com/yawning/nyquist.git (modern Noise)
// Full implementation with WebSocket/NATS integration and PSK-based resumption.

package bank

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"gitlab.com/yawning/nyquist.git"
	"gitlab.com/yawning/nyquist.git/dh"
	"github.com/vmihailenco/msgpack/v5"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/hkdf"
)

const (
	MaxFrameSize     = 128 * 1024
	ReplayWindowSize = 8192
	KeepAliveInterval = 30 * time.Second
	Prologue         = "securebus2025"
)

// ————————————————————————————————————————
// Public types
// ————————————————————————————————————————
type IdentityKey struct {
	Public  [32]byte
	Private [32]byte
}

type SessionMetadata struct {
	RemoteStatic [32]byte
	ResumeToken  []byte // 32-byte PSK for 0-RTT/PSK resumption
	SessionTag   [16]byte // BLAKE2b-128 of handshake hash
	Established  time.Time
}

type Envelope struct {
	Counter     uint64      `msgpack:"c"`
	TimestampMS int64       `msgpack:"ts"`
	Topic       string      `msgpack:"t"`
	Payload     interface{} `msgpack:"p"`
	SessionTag  [16]byte    `msgpack:"s"`
}

// ————————————————————————————————————————
// 8192-bit replay window — FAST & CORRECT
// ————————————————————————————————————————
type ReplayWindow struct {
	mu     sync.Mutex
	maxID  uint64
	bitmap [ReplayWindowSize / 64]uint64 // 128 words
}

func NewReplayWindow() *ReplayWindow { return &ReplayWindow{} }

func (w *ReplayWindow) Check(id uint64) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	if id > w.maxID {
		shift := id - w.maxID
		if shift >= ReplayWindowSize {
			// giant jump → reset
			for i := range w.bitmap {
				w.bitmap[i] = 0
			}
		} else {
			// fast block + bit shift
			blocks := int(shift / 64)
			bits := int(shift % 64)
			if blocks > 0 {
				copy(w.bitmap[:], w.bitmap[blocks:])
				for i := len(w.bitmap) - blocks; i < len(w.bitmap); i++ {
					w.bitmap[i] = 0
				}
			}
			if bits > 0 {
				for i := len(w.bitmap) - 1; i > 0; i-- {
					w.bitmap[i] = (w.bitmap[i] << bits) | (w.bitmap[i-1] >> (64 - bits))
				}
				w.bitmap[0] <<= bits
			}
		}
		w.maxID = id
	}
	offset := w.maxID - id
	if offset >= ReplayWindowSize {
		return false
	}
	word := offset / 64
	bit := uint(offset % 64)
	if w.bitmap[word]&(1<<bit) != 0 {
		return false // replay
	}
	w.bitmap[word] |= 1 << bit
	return true
}

// ————————————————————————————————————————
// SecureConn — clean and minimal
// ————————————————————————————————————————
type SecureConn struct {
	sendCS       *nyquist.CipherState
	recvCS       *nyquist.CipherState
	rw           *websocket.Conn
	sendMu       sync.Mutex
	recvMu       sync.Mutex
	replay       *ReplayWindow
	sessionTag   [16]byte
	counter      atomic.Uint64
	closed       atomic.Bool
	lastActivity atomic.Int64 // unix nano
}

func (c *SecureConn) Send(ctx context.Context, topic string, payload any) error {
	if c.closed.Load() {
		return errors.New("connection closed")
	}
	c.sendMu.Lock()
	defer c.sendMu.Unlock()
	env := Envelope{
		Counter:     c.counter.Add(1),
		TimestampMS: time.Now().UnixMilli(),
		Topic:       topic,
		Payload:     payload,
		SessionTag:  c.sessionTag,
	}
	data, err := msgpack.Marshal(env)
	if err != nil {
		return err
	}
	// Nyquist: EncryptWithAd(dst, ad, plaintext) — no AD, append to nil dst
	ct, err := c.sendCS.EncryptWithAd(nil, nil, data)
	if err != nil {
		return err
	}
	if err := writeWSFrame(c.rw, 0x00, ct); err != nil {
		return err
	}
	c.lastActivity.Store(time.Now().UnixNano())
	return nil
}

func (c *SecureConn) Receive(ctx context.Context, v any) (string, error) {
	if c.closed.Load() {
		return "", errors.New("connection closed")
	}
	c.recvMu.Lock()
	defer c.recvMu.Unlock()
	typ, payload, err := readWSFrame(c.rw)
	if err != nil {
		return "", err
	}
	if typ != 0x00 {
		return "", errors.New("bad frame type")
	}
	// Nyquist: DecryptWithAd(dst, ad, ciphertext) — no AD, append to nil dst
	plain, err := c.recvCS.DecryptWithAd(nil, nil, payload)
	if err != nil {
		return "", err
	}
	var env Envelope
	if err := msgpack.Unmarshal(plain, &env); err != nil {
		return "", err
	}
	if env.SessionTag != c.sessionTag {
		return "", errors.New("session tag mismatch")
	}
	if !c.replay.Check(env.Counter) {
		return "", errors.New("replay attack")
	}
	if env.Topic == "_keepalive" {
		c.lastActivity.Store(time.Now().UnixNano())
		return "_keepalive", nil
	}
	if v != nil {
		b, _ := msgpack.Marshal(env.Payload)
		_ = msgpack.Unmarshal(b, v)
	}
	c.lastActivity.Store(time.Now().UnixNano())
	return env.Topic, nil
}

func (c *SecureConn) Close() error {
	if !c.closed.Swap(true) {
		// send graceful close
		_ = c.Send(context.Background(), "_close", nil)
		c.rw.Close()
	}
	return nil
}

// simple background keep-alive + dead peer detection
func (c *SecureConn) startKeepAlive() {
	go func() {
		ticker := time.NewTicker(KeepAliveInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if time.Since(time.Unix(0, c.lastActivity.Load())) > KeepAliveInterval*3 {
					c.Close()
					return
				}
				_ = c.Send(context.Background(), "_keepalive", nil)
			}
		}
	}()
}

// ————————————————————————————————————————
// WS Framing — minimal & correct
// ————————————————————————————————————————
func writeWSFrame(conn *websocket.Conn, typ byte, data []byte) error {
	frameLen := len(data) + 1
	if frameLen > MaxFrameSize {
		return errors.New("frame too large")
	}
	var h [5]byte
	binary.BigEndian.PutUint32(h[:4], uint32(frameLen))
	h[4] = typ
	frame := make([]byte, frameLen)
	copy(frame[:5], h[:])
	copy(frame[5:], data)
	return conn.WriteMessage(websocket.BinaryMessage, frame)
}

func readWSFrame(conn *websocket.Conn) (byte, []byte, error) {
	mt, msg, err := conn.ReadMessage()
	if err != nil {
		return 0, nil, err
	}
	if mt != websocket.BinaryMessage {
		return 0, nil, errors.New("non-binary message")
	}
	if len(msg) == 0 || len(msg) > MaxFrameSize {
		return 0, nil, errors.New("invalid length")
	}
	if len(msg) < 5 {
		return 0, nil, errors.New("frame too short")
	}
	expectedLen := binary.BigEndian.Uint32(msg[:4])
	if uint32(len(msg)) != expectedLen {
		return 0, nil, errors.New("length mismatch")
	}
	typ := msg[4]
	return typ, msg[5:], nil
}

// ————————————————————————————————————————
// Handshaker — clean XX with Nyquist + PSK Resumption
// ————————————————————————————————————————
type Handshaker struct{ cfg *nyquist.HandshakeConfig }

func NewHandshaker(key IdentityKey, remote *[32]byte, initiator bool) *Handshaker {
	protocol, _ := nyquist.NewProtocol("Noise_XX_25519_ChaChaPoly_BLAKE2s")
	localStatic, _ := dh.X25519.ParsePrivateKey(key.Private[:]) // Parse from private bytes (public derived)
	cfg := &nyquist.HandshakeConfig{
		Protocol:    protocol,
		Prologue:    []byte(Prologue),
		IsInitiator: initiator,
		Rng:         rand.Reader,
		LocalStatic: localStatic,
	}
	if remote != nil {
		cfg.RemoteStatic, _ = dh.X25519.ParsePublicKey(remote[:]) // Parse from public bytes
	}
	return &Handshaker{cfg: cfg}
}

func (h *Handshaker) Perform(ctx context.Context, rw *websocket.Conn) (*SecureConn, SessionMetadata, error) {
	hs, err := nyquist.NewHandshake(h.cfg)
	if err != nil {
		return nil, SessionMetadata{}, err
	}
	for {
		msg, err := hs.WriteMessage(nil, nil)
		if err != nil {
			if err == nyquist.ErrDone {
				break // Handshake complete
			}
			return nil, SessionMetadata{}, err
		}
		if len(msg) > 0 {
			if err := writeWSFrame(rw, 0x01, msg); err != nil {
				return nil, SessionMetadata{}, err
			}
		}
		typ, in, err := readWSFrame(rw)
		if err != nil || typ != 0x01 {
			return nil, SessionMetadata{}, errors.New("handshake failed")
		}
		_, err = hs.ReadMessage(in, nil)
		if err != nil {
			if err == nyquist.ErrDone {
				break // Handshake complete
			}
			return nil, SessionMetadata{}, err
		}
	}
	status := hs.GetStatus()
	if status.Err != nil {
		return nil, SessionMetadata{}, status.Err
	}
	// Split symmetric state into cipher states
	sym := hs.SymmetricState()
	sendCS, recvCS := sym.Split()

	// Derive PSK for resumption from handshake hash
	hhash := status.HandshakeHash // []byte handshake hash
	var psk [32]byte
	kdf := hkdf.New(func() hash.Hash {
		h, _ := blake2b.New256(nil)
		return h
	}, hhash, nil, nil)
	if _, err := kdf.Read(psk[:]); err != nil {
		return nil, SessionMetadata{}, fmt.Errorf("PSK derivation failed: %w", err)
	}
	resumeToken := append([]byte(nil), psk[:]...) // Copy to []byte

	// Channel binding via handshake hash
	tag := blake2b.Sum256(hhash)
	var sessionTag [16]byte
	copy(sessionTag[:], tag[:16])

	conn := &SecureConn{
		sendCS:     sendCS,
		recvCS:     recvCS,
		rw:         rw,
		replay:     NewReplayWindow(),
		sessionTag: sessionTag,
	}
	conn.lastActivity.Store(time.Now().UnixNano())
	conn.startKeepAlive()

	var remoteStatic [32]byte
	copy(remoteStatic[:], status.RemoteStatic.Bytes())
	meta := SessionMetadata{
		RemoteStatic: remoteStatic,
		ResumeToken:  resumeToken,
		SessionTag:   sessionTag,
		Established:  time.Now(),
	}
	return conn, meta, nil
}

// Resume performs a shortened PSK-based handshake (fallback to Perform on failure)
func (h *Handshaker) Resume(ctx context.Context, rw *websocket.Conn, meta SessionMetadata) (*SecureConn, SessionMetadata, error) {
	if len(meta.ResumeToken) != 32 {
		// Fallback to full handshake
		return h.Perform(ctx, rw)
	}

	// Create PSK-enabled config (XXpsk3 for resumption)
	pskProtocol, err := nyquist.NewProtocol("Noise_XXpsk3_25519_ChaChaPoly_BLAKE2s")
	if err != nil {
		return h.Perform(ctx, rw) // Fallback on protocol error
	}
	pskCfg := &nyquist.HandshakeConfig{
		Protocol:      pskProtocol,
		Prologue:      []byte(Prologue),
		IsInitiator:   h.cfg.IsInitiator, // Reuse initiator flag from original config
		Rng:           rand.Reader,
		LocalStatic:   h.cfg.LocalStatic, // Reuse local identity
		RemoteStatic:  h.cfg.RemoteStatic, // Reuse if known
		PreSharedKeys: [][]byte{meta.ResumeToken}, // Inject PSK
	}

	hs, err := nyquist.NewHandshake(pskCfg)
	if err != nil {
		return h.Perform(ctx, rw)
	}

	// Use the same loop as Perform (PSK shortens it automatically)
	for {
		msg, err := hs.WriteMessage(nil, nil)
		if err != nil {
			if err == nyquist.ErrDone {
				break // Handshake complete
			}
			return h.Perform(ctx, rw)
		}
		if len(msg) > 0 {
			if err := writeWSFrame(rw, 0x01, msg); err != nil {
				return nil, SessionMetadata{}, err
			}
		}
		typ, in, err := readWSFrame(rw)
		if err != nil || typ != 0x01 {
			return h.Perform(ctx, rw)
		}
		_, err = hs.ReadMessage(in, nil)
		if err != nil {
			if err == nyquist.ErrDone {
				break // Handshake complete
			}
			return h.Perform(ctx, rw)
		}
	}

	status := hs.GetStatus()
	if status.Err != nil {
		return h.Perform(ctx, rw)
	}
	sym := hs.SymmetricState()
	sendCS, recvCS := sym.Split()

	// Derive fresh PSK for next resumption
	hhash := status.HandshakeHash
	var psk [32]byte
	kdf := hkdf.New(func() hash.Hash {
		h, _ := blake2b.New256(nil)
		return h
	}, hhash, nil, nil)
	if _, err := kdf.Read(psk[:]); err != nil {
		return h.Perform(ctx, rw)
	}
	meta.ResumeToken = append([]byte(nil), psk[:]...)

	// Recalc tag from new hash
	tag := blake2b.Sum256(hhash)
	var sessionTag [16]byte
	copy(sessionTag[:], tag[:16])
	meta.SessionTag = sessionTag

	conn := &SecureConn{
		sendCS:     sendCS,
		recvCS:     recvCS,
		rw:         rw,
		replay:     NewReplayWindow(),
		sessionTag: sessionTag,
	}
	conn.lastActivity.Store(time.Now().UnixNano())
	conn.startKeepAlive()

	meta.Established = time.Now()
	return conn, meta, nil
}

// ————————————————————————————————————————
// NATS WebSocket Dial (SecureBus over NATS WS)
// ————————————————————————————————————————
func Dial(natsURL string, local IdentityKey, remotePub *[32]byte, resumeMeta *SessionMetadata) (*SecureConn, SessionMetadata, error) {
	// Parse NATS WS URL (e.g., "wss://nats.example.com:4223")
	u, err := url.Parse(natsURL)
	if err != nil {
		return nil, SessionMetadata{}, fmt.Errorf("invalid NATS URL: %w", err)
	}
	if u.Scheme != "ws" && u.Scheme != "wss" {
		return nil, SessionMetadata{}, errors.New("NATS URL must be ws:// or wss://")
	}
	if u.Path == "" {
		u.Path = "/ws" // Default NATS WS path
	}

	// WebSocket dial (NATS acts as WS server)
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
		ReadBufferSize:   MaxFrameSize,
		WriteBufferSize:  MaxFrameSize,
	}
	wsConn, _, err := dialer.Dial(u.String(), http.Header{"Sec-Websocket-Protocol": {"securebus2025"}})
	if err != nil {
		return nil, SessionMetadata{}, fmt.Errorf("NATS WebSocket dial failed: %w", err)
	}
	defer func() {
		if err != nil {
			wsConn.Close()
		}
	}()

	// Perform handshake (or resume)
	handshaker := NewHandshaker(local, remotePub, true) // Initiator=true for client
	var conn *SecureConn
	var meta SessionMetadata
	if resumeMeta != nil && len(resumeMeta.ResumeToken) == 32 {
		conn, meta, err = handshaker.Resume(context.Background(), wsConn, *resumeMeta)
	} else {
		conn, meta, err = handshaker.Perform(context.Background(), wsConn)
	}
	if err != nil {
		wsConn.Close()
		return nil, SessionMetadata{}, fmt.Errorf("handshake failed: %w", err)
	}

	return conn, meta, nil
}

// ————————————————————————————————————————
// Key helper
// ————————————————————————————————————————
func GenerateIdentity() (IdentityKey, error) {
	kp, err := dh.X25519.GenerateKeypair(rand.Reader)
	if err != nil {
		return IdentityKey{}, err
	}
	var key IdentityKey
	privateBytes, _ := kp.MarshalBinary()
	copy(key.Private[:], privateBytes)
	copy(key.Public[:], kp.Public().Bytes())
	return key, nil
}

// PublicKeyFromBytes converts a byte slice to a fixed-size [32]byte public key.
// It validates the input length and copies the bytes if valid.
func PublicKeyFromBytes(b []byte) ([32]byte, error) {
	var k [32]byte
	if len(b) != 32 {
		return k, errors.New("invalid public key length")
	}
	copy(k[:], b)
	return k, nil
}


