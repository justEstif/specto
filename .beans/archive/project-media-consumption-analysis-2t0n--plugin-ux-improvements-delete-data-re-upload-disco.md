---
# project-media-consumption-analysis-2t0n
title: 'Plugin UX improvements: delete data, re-upload, disconnect modal'
status: completed
type: feature
priority: normal
created_at: 2026-03-13T20:59:26Z
updated_at: 2026-03-13T21:08:55Z
---

## Tasks

- [x] Add SQL query: delete media items by user+platform
- [ ] Add SQL query: delete sync log entries by user+plugin
- [ ] Regenerate sqlc
- [ ] Add handler: DELETE /api/v1/plugins/{plugin}/data
- [ ] Update plugin card template: hide Sync Now for file-upload plugins
- [ ] Update available plugins list: stop hiding connected plugins
- [ ] Add disconnect modal with disconnect-only and disconnect+delete options
- [ ] Test build compiles

## Summary of Changes

All 8 tasks completed:
- Added DeleteMediaItemsByPlatform and DeleteSyncLogsByPlugin SQL queries
- Added DeleteByPlatform to MediaItemStore and DeleteByPlugin to SyncLogStore interfaces + implementations
- Added DELETE /api/v1/plugins/{plugin}/data handler that wipes media items, sync logs, credentials, and resets state
- File-import plugins now show Re-upload instead of Sync Now when connected
- Connected plugins remain visible in the Available section
- Disconnect button opens a modal with two options: disconnect-only (keep data) or disconnect+delete (wipe everything)
- Fixed pre-existing mock interface mismatches for filtered insights methods
- All tests pass
