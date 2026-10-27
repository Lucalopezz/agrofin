package migrations

import "embed"

// Files expõe os arquivos .sql incorporados ao executável por //go:embed.
// O goose os lê como um sistema de arquivos virtual, sem buscá-los no contêiner.
// Ao adicionar uma migração, é preciso recompilar; go run faz isso ao notar a mudança.
//
//go:embed *.sql
var Files embed.FS
