package assets

import (
	"embed"
)

//go:embed logos/* font/* scripts/*
var EmbeddedFiles embed.FS
