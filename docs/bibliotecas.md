# bibliotecas usadas

Este documento descreve as bibliotecas usadas diretamente no código Go atual. As versões das dependências externas ficam em [`apps/api/go.mod`](../apps/api/go.mod), e o `go.sum` registra os hashes usados pelo Go para verificar os módulos baixados. Ainda não há ORM: o esquema está em migrações SQL.

## biblioteca padrão do go

| pacote | uso no projeto |
| --- | --- |
| [`embed`](https://pkg.go.dev/embed) | Incorpora os arquivos `*.sql` ao executável de migração. `migrations.Files` expõe esses arquivos ao goose sem leitura do disco durante a execução. Uma nova migração exige recompilar; `go run` faz isso ao executar novamente. |
| [`database/sql`](https://pkg.go.dev/database/sql) | Fornece `*sql.DB` ao migrador. `sql.Open` prepara o acesso e `PingContext` confirma a conexão. |
| [`context`](https://pkg.go.dev/context) | Limita o tempo de conexão e de execução das migrações. |
| [`testing`](https://pkg.go.dev/testing) | Define os testes e subtestes executados por `go test`. |
| [`net/http`](https://pkg.go.dev/net/http) e [`encoding/json`](https://pkg.go.dev/encoding/json) | Servem as rotas atuais da API e escrevem respostas JSON. |

## dependências externas

| módulo | uso no projeto |
| --- | --- |
| [`github.com/pressly/goose/v3`](https://github.com/pressly/goose) | Lê as migrações SQL, aplica as versões pendentes em ordem e registra o histórico na tabela `goose_db_version`. O comando `cmd/migrate` oferece `up`, `status` e `down`. |
| [`github.com/jackc/pgx/v5`](https://pkg.go.dev/github.com/jackc/pgx/v5) | Conecta ao PostgreSQL. `stdlib` registra o driver `"pgx"` para o `database/sql` usado pelo migrador; `pgxpool` cria o pool usado na inicialização da API. |
| [`github.com/stretchr/testify`](https://github.com/stretchr/testify) | Usado apenas nos testes, junto de `testing`. `assert` registra uma falha e permite continuar; `require` interrompe o teste atual quando os próximos passos dependem da verificação. O substituto do repositório continua sendo uma função simples, sem `testify/mock`. |

As demais entradas marcadas como `// indirect` no `go.mod` são dependências internas desses módulos. Não são chamadas diretamente pelo código do agrofin.

## ferramenta de desenvolvimento

| ferramenta | uso no projeto |
| --- | --- |
| [`github.com/air-verse/air`](https://github.com/air-verse/air) | Observa mudanças nos arquivos Go montados no contêiner `api`, recompila `cmd/api` e reinicia o processo. É instalado em versão fixa pelo `Dockerfile.dev`, sem entrar nas dependências da aplicação em `go.mod`. |

## frontend

A base React em `apps/web` usa React e React DOM, TypeScript, Vite com plugin React, Tailwind CSS com plugin Vite e Oxlint para análise estática. Os componentes shadcn são copiados para `shared/components/ui`; o botão usa Radix Slot e class-variance-authority. `clsx` e `tailwind-merge` compõem classes; `lucide-react` fornece os ícones. Versões e integridade ficam em `apps/web/package.json` e `pnpm-lock.yaml`. Veja a [estrutura do frontend](../apps/web/README.md).
