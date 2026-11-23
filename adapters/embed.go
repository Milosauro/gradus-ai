package adapters

import "embed"

// FS embeds the official declarative adapter manifests shipped with gradusctl.
//
//go:embed */adapter.yaml
var FS embed.FS
