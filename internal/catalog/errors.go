package catalog

import "errors"

// ErrReadOnly is returned by SaveSolution when the catalog has no database
// behind it. The screen turns this into a visible message rather than letting
// a save look like it worked.
var ErrReadOnly = errors.New("no database available: your work cannot be saved")
