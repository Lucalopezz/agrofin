# Bibliotecas e ferramentas

As principais dependências e seus usos no projeto são:

| tecnologia | uso |
| --- | --- |
| React e React DOM | Interface da aplicação. |
| TypeScript e Vite | Desenvolvimento, verificação de tipos e build do frontend. |
| Tailwind CSS e shadcn | Estilos e componentes de interface. |
| Oxlint | Análise estática do frontend. |
| pgx | Conexão da API Go com PostgreSQL. |
| goose | Execução e histórico das migrações SQL. |
| Testify | Verificações nos testes Go. |
| Air | Recompilação e reinício da API durante o desenvolvimento. |

O backend também usa a biblioteca padrão Go, incluindo `net/http` para HTTP e `encoding/json` para JSON. Não há ORM: o esquema é definido em SQL.

## Versões

As versões do backend estão em [go.mod](../apps/api/go.mod), as do frontend em [package.json](../apps/web/package.json) e as ferramentas de desenvolvimento nos Dockerfiles e em [compose.yaml](../compose.yaml). Os arquivos `go.sum` e `pnpm-lock.yaml` registram as dependências para instalações reproduzíveis.
