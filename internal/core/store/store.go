// Package store implements the repository layer that sits between the core
// domain logic and the database. It handles credential encryption, model
// conversion, and transactional boundaries.
//
// All store implementations wrap the sqlc-generated database.Queries and
// convert between core domain types (internal/core) and database models
// (internal/database).
//
// Store interfaces (MediaItemStore, PluginStateStore, SyncLogStore, TagStore,
// UserStore) and their associated domain types (PluginStateInfo, SyncLogEntry,
// SyncLogResult, MediaItemTagInfo, UserInfo) are defined in the core package
// to avoid import cycles.
package store
