# ADR-007: Credential Management and Secret Encryption

## Status
Accepted

## Context
Connectors are the bridge between Orchestrator workflows and external systems, and they require credentials to authenticate: API keys, OAuth2 access and refresh tokens, basic auth username/password pairs, bearer tokens, webhook signing secrets, and more. These credentials are tenant-owned, sensitive data that must be stored securely, accessed efficiently, and rotated without downtime.

The credential landscape is varied. Some credentials are static (API keys that rarely change), while others are refreshable (OAuth2 tokens with expiry). Some tenants may have dozens of credentials across multiple connector types. The system must support the full lifecycle: creation, encrypted storage, retrieval for workflow execution, rotation, and eventual deletion.

Foundry uses AWS Secrets Manager for infrastructure-level secrets (database passwords, service account keys). However, Orchestrator needs a general-purpose credential system for arbitrary user-provided credentials. Using AWS Secrets Manager for potentially thousands of per-tenant credentials would be cost-prohibitive (per-secret pricing) and introduce unnecessary AWS API latency into the hot path of every connector call.

## Decision
We adopt a two-layer credential storage architecture combining encrypted PostgreSQL storage with short-lived Valkey caching.

### Layer 1: Encrypted At-Rest in PostgreSQL

The `connector_credentials` table stores credentials as AES-256-GCM encrypted blobs. The master encryption key is loaded from an environment variable (in production, backed by AWS KMS). A `key_version` column on each row tracks which version of the master key was used for encryption, enabling zero-downtime key rotation.

```
connector_credentials
├── id (UUID, PK)
├── tenant_id (UUID, NOT NULL, FK)
├── connector_type (TEXT, NOT NULL)
├── credential_type (TEXT, NOT NULL)  -- api_key, oauth2, basic_auth, bearer, webhook_secret
├── encrypted_data (BYTEA, NOT NULL)  -- AES-256-GCM ciphertext
├── key_version (INT, NOT NULL)       -- encryption key version
├── metadata (JSONB)                  -- non-sensitive: scopes, expiry hints
├── created_at (TIMESTAMPTZ)
├── updated_at (TIMESTAMPTZ)
└── rotated_at (TIMESTAMPTZ)
```

### Layer 2: Short-Lived Valkey Cache (5-Minute TTL)

To prevent repeated decryption on every connector call, decrypted credentials are cached in Valkey with a strict 5-minute TTL. The cache key format is `cred:{tenant_id}:{credential_id}`. This limits the exposure window while eliminating decryption as a bottleneck during burst activity.

### Credential Lifecycle

**Storage flow (user sets credential):**
1. User calls `SetCredential` RPC with credential type and plaintext value
2. Server validates credential type and required fields
3. Encrypt plaintext with current master key version using AES-256-GCM
4. Store encrypted blob in `connector_credentials` with `key_version`
5. Invalidate any existing Valkey cache entry for this credential

**Retrieval flow (activity needs credential):**
1. Worker requests credential by `credential_id` and `tenant_id`
2. Check Valkey cache `cred:{tenant_id}:{credential_id}`
3. Cache hit: return cached plaintext credential
4. Cache miss: load encrypted row from PostgreSQL, decrypt with appropriate key version, cache in Valkey with 5-minute TTL, return plaintext

**Key rotation flow:**
1. Deploy new master key with incremented version number
2. New credentials are encrypted with the new key version
3. Background job iterates all rows, decrypts with old key, re-encrypts with new key, updates `key_version`
4. Old key is retained in the key map until all rows are migrated
5. Once migration is complete, old key can be removed from configuration

## Alternatives Considered

| Option | Pros | Cons |
|--------|------|------|
| Encrypted PostgreSQL + Valkey cache (chosen) | No external dependency; zero-downtime key rotation; limited cache exposure window; clear audit trail via database | Must implement crypto correctly; master key management is our responsibility |
| AWS Secrets Manager | Managed service; automatic rotation support; audit via CloudTrail | AWS-specific lock-in; per-secret pricing ($0.40/secret/month) impractical at scale; API latency on every retrieval; not suitable for thousands of user credentials |
| HashiCorp Vault | Industry standard; dynamic secrets; comprehensive audit logging | Additional infrastructure to deploy and manage; operational overhead; overkill for user-provided credential storage |
| Plaintext in PostgreSQL | Simple implementation; no crypto complexity | Unacceptable security posture; database breach exposes all credentials; fails compliance requirements |
| pgcrypto extension | Encryption at database level; no application crypto code | Encryption key exposed in SQL queries and logs; key management through database functions is awkward; less control over algorithm selection |

## Rationale
1. User-provided credentials are fundamentally different from infrastructure secrets: there can be thousands per platform, making per-secret pricing models impractical.
2. AES-256-GCM is the industry standard for authenticated encryption, providing both confidentiality and integrity verification.
3. Key versioning enables zero-downtime rotation by allowing old and new keys to coexist during the migration window.
4. A short-lived 5-minute cache prevents decryption from becoming a bottleneck during workflow execution bursts while limiting the window of exposure for cached plaintext.
5. Storing credentials in PostgreSQL alongside other tenant data keeps the operational model simple and backup/restore procedures unified.

## Consequences

**Positive:**
- No external dependency beyond the existing PostgreSQL and Valkey infrastructure
- Key rotation can happen without downtime or service interruption
- 5-minute cache TTL limits the exposure window for decrypted credentials
- Clear audit trail through database timestamps and credential metadata
- Consistent with the self-contained operational model of the platform

**Negative:**
- Cryptographic implementation must be correct; subtle bugs can compromise all credentials
- Master key management is our responsibility (mitigated by KMS-backing in production)
- Decrypted credentials must never appear in logs, traces, or error messages; requires discipline across the codebase
- Cache invalidation on credential update must be reliable to prevent stale credential usage

## Implementation Notes

### Encryption Package

```go
// pkg/crypto/encrypt.go

package crypto

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "fmt"
    "io"
)

// Encrypt encrypts plaintext using AES-256-GCM with the given master key.
// Returns ciphertext with nonce prepended.
func Encrypt(masterKey []byte, keyVersion int, plaintext []byte) ([]byte, int, error) {
    block, err := aes.NewCipher(masterKey)
    if err != nil {
        return nil, 0, fmt.Errorf("create cipher: %w", err)
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, 0, fmt.Errorf("create GCM: %w", err)
    }

    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return nil, 0, fmt.Errorf("generate nonce: %w", err)
    }

    ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
    return ciphertext, keyVersion, nil
}

// Decrypt decrypts ciphertext using the appropriate master key version.
// Expects nonce prepended to ciphertext.
func Decrypt(masterKeys map[int][]byte, keyVersion int, ciphertext []byte) ([]byte, error) {
    masterKey, ok := masterKeys[keyVersion]
    if !ok {
        return nil, fmt.Errorf("unknown key version: %d", keyVersion)
    }

    block, err := aes.NewCipher(masterKey)
    if err != nil {
        return nil, fmt.Errorf("create cipher: %w", err)
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, fmt.Errorf("create GCM: %w", err)
    }

    nonceSize := gcm.NonceSize()
    if len(ciphertext) < nonceSize {
        return nil, fmt.Errorf("ciphertext too short")
    }

    nonce, encrypted := ciphertext[:nonceSize], ciphertext[nonceSize:]
    plaintext, err := gcm.Open(nil, nonce, encrypted, nil)
    if err != nil {
        return nil, fmt.Errorf("decrypt: %w", err)
    }

    return plaintext, nil
}
```

### Future Enhancements
- **OAuth2 token refresh:** Background job refreshes tokens before expiry, updating encrypted storage and cache.
- **Credential sharing:** Allow tenants to share read-only credentials across workspaces within the same organization.
- **Rotation alerts:** Notify tenants when credentials approach recommended rotation age.
- **External vault integration:** Optional integration with HashiCorp Vault or AWS Secrets Manager for enterprise tenants.
- **Credential templates:** Pre-built credential schemas per connector type with validation rules.
