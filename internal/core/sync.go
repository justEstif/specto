package core

// SyncResult is returned by SourcePlugin.Sync().
// It supports partial success and incremental sync via cursors.
type SyncResult struct {
	Items      []MediaItem  // items fetched (may be partial on error)
	NextCursor string       // opaque cursor for incremental sync — core stores and replays
	HasMore    bool         // true if pagination was not exhausted
	Err        *PluginError // nil on full success
}
