// Package detection implements Phase 0 death-candidate inference from parsed
// combat-log events: Mythic+ run boundaries, boss encounter context, recent
// damage buffers, and ranked likely causes. It does not upload data or persist
// state beyond an in-memory scan.
package detection
