package api

import "embed"

//go:embed openapi/*.yaml
var OpenAPIFS embed.FS

//go:embed docs/index.html
var DocsFS embed.FS
