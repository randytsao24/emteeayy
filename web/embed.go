// Package web contains the embedded frontend files
package web

import "embed"

//go:embed index.html
var FS embed.FS
