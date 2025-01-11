package content

import "embed"

// Makes all files from currand all subdirectories available as a Go module.
// Needed for serving the files online.

//go:embed **/*
var ContentFiles embed.FS 
