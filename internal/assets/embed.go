package assets

import (
	"embed"
)

//go:embed scripts/* bashrc.d/* r/*
var FS embed.FS
