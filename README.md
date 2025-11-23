# SecureBus 2025 - Secure Messaging Transport

A secure, high-performance messaging transport protocol providing mutual authentication, forward secrecy, and zero-round-trip capability.

## Overview

SecureBus 2025 is a secure transport layer that enables authenticated, encrypted communication between distributed nodes. Built on the Noise Protocol Framework, it provides enterprise-grade security with minimal overhead, designed specifically for integration with NATS JetStream and MessagePack serialization.

## Features

- 🔐 **Mutual Authentication** - Both parties verify identities using X25519 keypairs
- 🗝️ **Forward Secrecy** - Ephemeral keys ensure past sessions remain secure
- ⚡ **0-RTT Capability** - Optional zero-round-trip resumption for low latency
- 🛡️ **Replay Protection** - Sliding window mechanism prevents message replay attacks
- 🔄 **Session Resumption** - Efficient session recovery without full handshake
- 📦 **Lightweight** - ~300-400 lines of Go code

## Quick Start

### Prerequisites

- Go 1.21+
- NATS JetStream
- Vault or GitOps for key management

### Installation

```go
import "github.com/your-org/securebus"
```

### Basic Usage

```go
// Load identity keys
config := securebus.HandshakeConfig{
    Local:   identityKey,
    Prologue: []byte("securebus2025-v1"),
}

// As initiator
conn, metadata, err := config.Initiate(remotePublicKey, transport)
if err != nil {
    return err
}

// Send secure message
err = conn.Send(&securebus.Envelope{
    MessageID:  1,
    Timestamp:  time.Now().UnixMilli(),
    Topic:      "orders.create",
    Payload:    orderRequest,
})

// Receive message
var envelope securebus.Envelope
err = conn.Receive(&envelope)
```

## Protocol Specification

### Cryptographic Foundation

- **Identity Keys**: X25519 keypairs (32 bytes each)
- **Handshake Pattern**: `Noise_XXfallback`
- **Cipher Suite**: ChaCha20-Poly1305 + BLAKE2s
- **Prologue**: `"securebus2025-v1"`

### Message Framing

```
+-----------------+----------------------+---------------------+
| 4-byte length N | 1-byte type          | N-1 bytes payload   |
| big-endian      | (0=data,1=control)   | encrypted           |
+-----------------+----------------------+---------------------+
```

### Message Envelope (MsgPack)

```go
type Envelope struct {
    MessageID  uint64 `msgpack:"message_id"`
    Timestamp  int64  `msgpack:"timestamp_ms,omitempty"`
    Topic      string `msgpack:"topic"`
    Payload    any    `msgpack:"payload"`
}
```

## Key Management

### Vault Integration (Recommended)

```bash
# Node identity storage
secret/securebus/nodes/<node-id>/identity
secret/securebus/nodes/<peer-id>/pub
```

### GitOps Alternative

```
keys/
  ├── nodeA.pub
  ├── nodeB.pub
  └── nodeC.pub
```

Private keys stored as Kubernetes Secrets or SOPS-encrypted files.

## Security Model

### Replay Protection

- 64-bit monotonic message IDs
- 2048-entry sliding window
- Session-level replay tokens

### Thread Safety

- Separate mutexes for send/receive operations
- CipherState is not thread-safe
- SecureConn provides safe concurrent access

## Integration with NATS

SecureBus works seamlessly with NATS JetStream over WebSocket:

```go
// WebSocket transport over NATS
transport, err := nats.Connect("wss://nats-host:443/securebus")
```

## API Reference

### Core Types

```go
type SecureConn interface {
    Send(any) error      // Thread-safe send
    Receive(any) error   // Thread-safe receive
    Close() error        // Clean shutdown
}

type Handshaker interface {
    Initiate() (SecureConn, SessionMetadata, error)
    Respond() (SecureConn, SessionMetadata, error)
    Resume(SessionMetadata) (SecureConn, error)
}
```

## Implementation Guardrails

### Required Practices

✅ Use Noise library's built-in key generation  
✅ Validate prologue consistently  
✅ Apply replay protection for 0-RTT  
✅ Zero private keys in memory when done  
✅ Use frame size limits (64KB max)  

### Security Anti-Patterns

❌ Never expose private keys in logs  
❌ Don't skip mutex protection for CipherState  
❌ Avoid reusing SessionMetadata across peers  
❌ Don't trust frame lengths without validation  

## Example: Complete Workflow

```go
// Server (Responder)
func handleConnection(ws net.Conn) {
    conn, metadata, err := handshaker.Respond()
    if err != nil {
        log.Printf("Handshake failed: %v", err)
        return
    }
    
    for {
        var envelope securebus.Envelope
        if err := conn.Receive(&envelope); err != nil {
            break
        }
        
        // Process application message
        handleMessage(envelope.Topic, envelope.Payload)
    }
}

// Client (Initiator)  
func sendOrder(remotePubKey [32]byte, order Order) error {
    conn, _, err := handshaker.Initiate(remotePubKey, wsTransport)
    if err != nil {
        return err
    }
    defer conn.Close()
    
    return conn.Send(&securebus.Envelope{
        MessageID: 1,
        Topic:     "orders.create",
        Payload:   order,
    })
}
```

## Monitoring & Operations

### Key Metrics

- Handshake success rate
- Session resumption rate
- Message throughput
- Replay detection counts

### Logging

```go
// Structured logging for security events
logger.Info("securebus_handshake_completed",
    "remote_key", hex.EncodeToString(metadata.RemoteStatic[:8]),
    "session_type", metadata.ResumeToken != nil ? "resumed" : "full",
)
```

## Troubleshooting

### Common Issues

1. **Handshake Fails**
   - Verify prologue matches exactly
   - Check key clamping
   - Validate remote public keys

2. **Replay Errors**
   - Check system clock synchronization
   - Verify message ID sequencing
   - Inspect sliding window state

3. **Performance Issues**
   - Monitor NATS backpressure
   - Check session resumption rate
   - Profile cipher operations

## Contributing

Please read our [Security Guidelines](SECURITY.md) before contributing to this security-critical codebase.

## License

Proprietary - Internal Use Only

---

**Version**: 1.0  
**Status**: Production Ready  
**Cryptographic Note**: This implementation uses vetted Noise Protocol libraries and does not contain custom cryptographic primitives.
