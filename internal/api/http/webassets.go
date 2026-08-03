package httpapi

import "embed"

//go:embed webdist/*
var WebAssets embed.FS
