# agrofin

Sistema de gestão da produção de limão, pensado para lançamentos rápidos no celular. O cliente, gerente da plantação, é o único usuário do sistema; funcionários são cadastrados apenas para relacionar colheitas e pagamentos.

## estado atual

O projeto está em desenvolvimento. A API Go responde em `GET /api/v1/health` e `GET /api/v1/employees` (ainda uma lista vazia). O PostgreSQL local e as migrações do modelo relacional estão prontos. Cadastro funcional, autenticação, criptografia dos valores financeiros, demais fluxos e frontend React ainda não foram implementados.

## início rápido

Para executar os serviços, instale Docker com Compose. Em uma nova cópia do repositório, crie o `.env` a partir do exemplo:

```bash
cp .env.example .env
```

Troque `POSTGRES_PASSWORD` no `.env` por uma senha hexadecimal longa antes de iniciar. Se o arquivo já existir, mantenha as credenciais atuais. Depois, rode:

```bash
docker compose up -d --wait
curl http://127.0.0.1:8080/api/v1/health
```

O Compose sobe PostgreSQL, aplica as migrações pendentes e inicia a API nessa ordem. Por padrão, a API fica em `127.0.0.1:8080` e o banco em `127.0.0.1:5432`; as portas podem ser alteradas no `.env`. O arquivo `.env` não é versionado.

## comandos úteis

| ação | comando |
| --- | --- |
| Ver o estado dos serviços | `docker compose ps` |
| Acompanhar os logs da API | `docker compose logs -f api` |
| Consultar as migrações aplicadas | `docker compose run --rm migrate go run ./cmd/migrate status` |
| Aplicar uma migração adicionada depois da primeira subida | `docker compose run --rm migrate` |
| Parar os serviços, mantendo os dados do banco | `docker compose down` |

Para verificar o código Go fora dos contêineres, use Go 1.27.3 e execute `go test ./...` dentro de `apps/api`, onde está o `go.mod`.

## organização

| caminho | responsabilidade |
| --- | --- |
| `apps/api/cmd/api` | Inicia a API HTTP e a conexão com o banco. |
| `apps/api/cmd/migrate` | Aplica, consulta e reverte migrações. |
| `apps/api/internal/funcionarios` | Rotas e futuro cadastro de funcionários. |
| `apps/api/internal/platform/postgres` | Cria e verifica o pool PostgreSQL da API. |
| `apps/api/migrations` | Migrações SQL incorporadas ao executável Go. |

`apps/web` é a aplicação React planejada e ainda não existe. O deploy unificado com a API também não foi validado. Veja [o contexto do projeto](CONTEXT.md) e as [orientações para agentes](AGENTS.md) antes de implementar novos fluxos.

## documentação

- [bibliotecas usadas](docs/bibliotecas.md)
- [modelo de dados e relacionamentos](docs/modelo-de-dados.md)
- [requisitos](docs/requisitos.md)
- [histórias de usuário](docs/historias-de-usuario.md)
- [critérios de aceite](docs/criterios-de-aceite.md)
- [fluxos mobile](docs/fluxos-mobile.md)
- [plano de implementação](docs/plano-de-implementacao.md)
- [skills externas e suas fontes](docs/skills-externas.md)

Na documentação, **cliente** significa a pessoa que administra a produção.
