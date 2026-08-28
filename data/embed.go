// Package data holds the seed problem set and title art that ship inside the
// binary.
//
// These files are embedded rather than read from disk so clicode runs from any
// working directory — a `go install`ed binary has no repo root to find them in.
// They are seed data, not the app's system of record: once the SQLite store
// lands, these are what a fresh install is populated from.
package data

import (
	"embed"
	"io/fs"
)

//go:embed problems/*.json art/*.txt
var files embed.FS

// Problems returns the embedded problem seed files, rooted so entries are named
// "1.json", "problems_list.json", and so on.
func Problems() fs.FS {
	sub, err := fs.Sub(files, "problems")
	if err != nil {
		// Unreachable: the embed directive guarantees this directory exists.
		panic(err)
	}
	return sub
}

// Art returns the built-in title art. The second value is false if the art file
// is missing or blank, so callers can fall back to their own banner.
func Art() (string, bool) {
	b, err := files.ReadFile("art/current.txt")
	if err != nil {
		return "", false
	}
	return string(b), true
}
