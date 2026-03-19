package migrations

import "embed"

// FS exposes SQL migration files to the application runtime.
//
//go:embed *.up.sql *.down.sql
var FS embed.FS
