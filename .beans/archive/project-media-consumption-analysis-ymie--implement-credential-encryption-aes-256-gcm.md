---
# project-media-consumption-analysis-ymie
title: Implement credential encryption (AES-256-GCM)
status: completed
type: task
priority: high
created_at: 2026-03-12T20:20:54Z
updated_at: 2026-03-12T20:34:34Z
parent: project-media-consumption-analysis-86vz
blocked_by:
    - project-media-consumption-analysis-3vdi
---

Create internal/core/store/crypto.go with Encrypt(plaintext, key) and Decrypt(ciphertext, key) using AES-256-GCM. The encryption key comes from an environment variable. Used by the store layer to encrypt/decrypt plugin credentials (OAuth tokens, API keys) before persisting to plugin_credentials table. Include unit tests with round-trip verification.

\n## Todo\n\n- [x] Create internal/core/store/ directory\n- [x] Implement crypto.go with Encrypt/Decrypt using AES-256-GCM\n- [x] Add unit tests (round-trip, wrong key, tampered ciphertext, empty input)\n- [x] Verify build passes

## Summary of Changes

Created `internal/core/store/crypto.go` with:

- `Encrypt(plaintext, hexKey)` — AES-256-GCM encryption, random nonce prepended to output
- `Decrypt(ciphertext, hexKey)` — AES-256-GCM decryption with authentication
- Key validation: hex-encoded, must decode to exactly 32 bytes

9 unit tests covering round-trip, nonce uniqueness, wrong key, tampered data, too-short input, empty plaintext, invalid key format, wrong key length, and 1MB payload.
