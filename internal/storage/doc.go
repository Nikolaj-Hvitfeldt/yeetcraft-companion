// Package storage persists companion capture state in a local SQLite database.
//
// The schema is companion-local only; it is not a replica of Yeetcraft PostgreSQL.
// Byte offsets advance in the same transaction as the events derived from those bytes.
package storage
