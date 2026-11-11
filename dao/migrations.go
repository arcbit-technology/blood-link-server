package dao

import "embed"

// migrationsFS embeds the SQL migrations into the binary so the server
// does not need the dao/migrations directory on disk at runtime.
//
//go:embed migrations/*.sql
var migrationsFS embed.FS
