---
# project-media-consumption-analysis-ymie
title: Implement credential encryption (AES-256-GCM)
status: todo
type: task
priority: high
created_at: 2026-03-12T20:20:54Z
updated_at: 2026-03-12T20:20:54Z
parent: project-media-consumption-analysis-86vz
blocked_by:
    - project-media-consumption-analysis-3vdi
---

Create internal/core/store/crypto.go with Encrypt(plaintext, key) and Decrypt(ciphertext, key) using AES-256-GCM. The encryption key comes from an environment variable. Used by the store layer to encrypt/decrypt plugin credentials (OAuth tokens, API keys) before persisting to plugin_credentials table. Include unit tests with round-trip verification.
