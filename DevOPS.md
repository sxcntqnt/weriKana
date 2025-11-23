# SecureBus 2025 - Operational Package

## 1. Developer Onboarding Guide

### Quick Start (15-20 minutes)

**What SecureBus Is:**
- Secure transport layer on top of NATS JetStream + WebSockets
- Uses Noise_XXfallback handshake with MsgPack envelopes
- Provides: Forward Secrecy, Mutual Authentication, Replay Protection, 0-RTT Resumption
- Typical latency: <80 μs

**Prerequisites:**
- Go 1.22+
- Docker + kubectl
- Access to Vault/GitOps repo
- Access to staging NATS

### Identity Keys for Development

Each developer gets:
```yaml
dev-identity:
  public: 32 bytes
  private: 32 bytes
```

**Storage:**
- Vault dev namespace, OR
- SOPS-encrypted GitOps repo

*Developers do NOT generate production keys.*

### Local Development Setup

**Project Structure:**
```
securebus-config/
    identity.key
    peers/
       serviceA.pub
       serviceB.pub
```

**Run:**
```bash
make dev-run
```

This spins up:
- Local NATS JetStream
- SecureBus handler stub
- WebSocket tunnel
- Test harness messages

### Monitoring & Logging

SecureBus emits structured logs:
- `securebus.session.established`
- `securebus.session.resumed`
- `securebus.msg.send`
- `securebus.msg.recv`
- `securebus.replay.drop`

### Integration Pattern

Developers only implement:
```go
handler.OnMessage(topic, payload)
handler.Send(topic, payload)
```

SecureBus handles all encryption/authentication automatically.

### Developer Security Responsibilities

- ✅ Never log identity private keys
- ✅ Never send unbounded payloads
- ✅ Use replay-safe Send() only
- ✅ Never open raw WebSocket connections
- ❌ Don't bypass SecureBus for "temporary debug"

---

## 2. GitOps Deployment Pattern

### Repository Structure
```
gitops/
  securebus/
    overlays/
      prod/
      staging/
      dev/
    base/
      deployment.yaml
      service.yaml
      configmap.yaml
      identity-secret.yaml
      peers-config.yaml
```

### Identity Key Distribution

**identity-secret.yaml** (SOPS-encrypted):
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: securebus-identity
type: Opaque
data:
  private: <base64>
  public: <base64>
```

**peers-config.yaml** (plaintext - public keys only):
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: securebus-peers
data:
  serviceA.pub: "<base64>"
  serviceB.pub: "<base64>"
```

### Deployment Configuration

**deployment.yaml:**
```yaml
containers:
  - name: app
    image: myapp:prod
    env:
      - name: SECUREBUS_PRIVATE_KEY
        valueFrom:
          secretKeyRef:
            name: securebus-identity
            key: private
      - name: SECUREBUS_PUBLIC_KEY
        valueFrom:
          secretKeyRef:
            name: securebus-identity
            key: public
      - name: SECUREBUS_PEERS_CONFIG
        value: "/etc/securebus/peers"
      - name: SECUREBUS_NATS_URL
        value: "wss://nats.prod.svc.cluster.local:443"
    volumeMounts:
      - name: peers-config
        mountPath: /etc/securebus/peers
```

### Key Rotation Workflow

1. Commit new identity public key to `peers/` directory
2. Rotate private key in Vault/SOPS
3. Rolling restart via ArgoCD
4. SecureBus automatically renegotiates sessions

*Rotation window: Every 90 days*

---

## 3. Formal Threat Model

### Assets Protected
- Identity private keys
- CipherState (session AEAD keys)
- Replay windows
- Session resumption tokens
- User message payloads

### Adversary Profiles
- External attacker with network access
- Malicious insider
- Compromised microservice
- Passive eavesdropper
- Nation-state with packet capture capability

### STRIDE Analysis

| Threat | Relevant? | Mitigation |
|--------|-----------|------------|
| **Spoofing** | ✅ Yes | Noise mutual authentication prevents impersonation |
| **Tampering** | ✅ Yes | AEAD prevents ciphertext modification |
| **Repudiation** | ❌ No | Not designed for legal non-repudiation |
| **Info Disclosure** | ✅ Yes | Noise AEAD + Forward Secrecy protects confidentiality |
| **Denial of Service** | ✅ Yes | Frame-size caps, handshake timeouts |
| **Elevation of Privilege** | ✅ Yes | Identity keys tightly scoped |

### MITRE ATT&CK Coverage

**Network Attacks:**
- Sniffing traffic → Noise protects
- MITM → Static key mutual authentication
- Replay → message_id + sliding window
- TLS downgrade → N/A (Noise runs inside wss)

**Host Attacks:**
- Private key exfiltration → Vault/KMS mitigation
- Memory scraping → Short-lived CipherState; zero on close

**Application Attacks:**
- Oversized payload → Size capped
- Concurrent cipher access → Mutex protected

---

## 4. Message Validation Checklist

### Application-Level Validation
- [ ] `Envelope.message_id` is monotonic
- [ ] `Envelope.timestamp_ms` within ±2 minutes
- [ ] `topic` is non-empty
- [ ] `payload` is valid MsgPack

### Transport-Level Validation
- [ ] Frame length ≤ 64 KB
- [ ] Type byte is 0 (data) or 1 (control)
- [ ] Noise decryption returns non-nil
- [ ] Frame structure valid

### Replay Protection
- [ ] `message_id > last_seen` OR inside window
- [ ] Replay window bit not already set
- [ ] Window updates atomically
- [ ] Sliding window size = 2048 entries

### Identity Validation
- [ ] Remote static key matches expected peer identity
- [ ] Prologue matches "securebus2025-v1"
- [ ] Noise handshake completes without fallback loops

### Concurrency Safety
- [ ] `sendState` protected by mutex
- [ ] `recvState` protected by mutex
- [ ] No concurrent writes to same WebSocket
- [ ] Thread-safe SecureConn implementation

---

## 5. Test Suite Specification

### A. Handshake Tests
1. **XXfallback Completion** - Verify successful handshake
2. **Resumption Success** - Session resume works correctly
3. **Invalid Resumption Token** - Failed resume falls back to full handshake
4. **Wrong Prologue** - Connection rejected with mismatched prologue
5. **Wrong Static Key** - Handshake fails with incorrect peer key

### B. CipherState Tests
1. **Concurrent Send** - No race conditions under load
2. **Concurrent Receive** - Thread-safe message processing
3. **Nonce Monotonic** - Nonce increments sequentially
4. **CipherState Cleanup** - No reuse after Close()

### C. Replay Tests
1. **Same Session Replay** - Duplicate messages rejected
2. **Cross-Session Replay** - Resumed sessions protected
3. **Out-of-Order Acceptance** - Messages within window accepted
4. **Out-of-Window Rejection** - Old messages properly dropped

### D. Frame Tests
1. **Oversized Frame** - Frames >64 KB rejected
2. **Zero-Length Frame** - Invalid frames rejected
3. **Malformed Length** - Connection safely closed
4. **Partial Frame** - Proper timeout handling

### E. Integration Tests
1. **End-to-End Delivery** - Send/receive envelope verification
2. **Topic Routing** - Correct message routing by topic
3. **NATS Reconnect** - Handshake resume on reconnection
4. **Load Testing** - No deadlocks under high load

### F. Adversarial Tests
1. **MITM Handshake Disruption** - Handshake fails safely when tampered
2. **Garbage Ciphertext** - Malformed data properly dropped
3. **Replay Flood** - System remains stable under replay attack
4. **Resource Exhaustion** - Memory/CPU limits respected

### Test Environment Requirements
```yaml
test-environment:
  nodes: 3+ node cluster
  load: 10K+ messages/second
  duration: 72-hour stability test
  tools: 
    - fuzzing harness
    - chaos engineering toolkit
    - performance profiling
```

---

## Security Compliance Checklist

### Pre-Production Validation
- [ ] All identity keys properly stored (Vault/SOPS)
- [ ] Peer public keys correctly distributed
- [ ] Frame size limits enforced
- [ ] Replay protection enabled
- [ ] Concurrency safety verified
- [ ] Memory zeroization implemented
- [ ] Logging excludes sensitive data

### Runtime Monitoring
- [ ] Handshake success rate >99.9%
- [ ] Replay detection alerts configured
- [ ] Session resumption rate tracked
- [ ] Latency percentiles monitored
- [ ] Error rate thresholds defined

### Incident Response
- [ ] Key compromise procedure documented
- [ ] Session revocation process defined
- [ ] Forensic logging enabled
- [ ] Rollback procedures tested

This operational package ensures SecureBus 2025 deployments maintain security, reliability, and performance across all environments.
