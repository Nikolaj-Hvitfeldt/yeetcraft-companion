// Package logwatcher polls combat-log files with resumable byte offsets.
//
// Offsets are authoritative; polling is only a hint. Truncation and rotation
// create a new file generation and restart from offset zero.
package logwatcher
