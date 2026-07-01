package cli

import "os"

// osGetenv wraps os.Getenv so tests can stub $EDITOR / $VISUAL without
// touching the real environment (edit.go delegates through this).
var osGetenv = os.Getenv
