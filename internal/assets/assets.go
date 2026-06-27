package assets

import (
	"embed"
)

//go:embed logos/* font/*
var EmbeddedFiles embed.FS
