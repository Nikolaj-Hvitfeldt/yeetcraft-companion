// Package parser incrementally reads and parses WoW combat-log lines.
//
// It preserves unknown records, applies V22-specific interpretation only
// while the format state is supported, and does not implement death inference.
package parser
