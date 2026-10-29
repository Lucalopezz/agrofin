# Agrofin

Sistema de gestão da produção de limão, pensado para lançamentos rápidos no celular. O cliente, gerente da plantação, é o único usuário do sistema; funcionários são cadastrados apenas para relacionar colheitas e pagamentos.

## Estado atual

O projeto está em desenvolvimento. A API Go responde em `GET /api/v1/health`, `GET /api/v1/employees` (ainda uma lista vazia) e `POST /api/v1/employees`, que responde `503` até a integração com o login. O PostgreSQL local e as migrações do modelo relacional estão prontos. Autenticação, listagem persistida, renomeação, desativação, criptografia dos valores financeiros e demais fluxos ainda não foram implementados. A base React + TypeScript, com Tailwind, shadcn e atualização automática pelo Docker, está disponível.

O cadastro separa HTTP no handler, validação e montagem da entidade no serviço e inserção no repositório PostgreSQL; veja a [organização do cadastro](docs/plano-de-implementacao.md#organização-interna-do-cadastro-de-funcionários).

## Início rápido

Para executar os serviços, instale Docker com Compose. Em uma nova cópia do repositório, crie o `.env` a partir do exemplo:

```bash
cp .env.example .env
```

Troque `POSTGRES_PASSWORD` no `.env` por uma senha hexadecimal longa antes de iniciar. Se o arquivo já existir, mantenha as credenciais atuais. Depois, rode:

```bash
docker compose up -d --wait
curl http://127.0.0.1:8080/api/v1/health
```

O Compose sobe PostgreSQL, aplica as migrações pendentes, inicia a API e depois o frontend em http://localhost:5173. Por padrão, a API fica em `127.0.0.1:8080` e o banco em `127.0.0.1:5432`; as portas podem ser alteradas no `.env`. O arquivo `.env` não é versionado.

### Desenvolvimento da api

No primeiro `up`, o Docker constrói a imagem de desenvolvimento com o Air. O serviço `api` observa os arquivos `.go`, `go.mod` e `go.sum` em `apps/api`: ao salvar uma alteração, o Air recompila e reinicia a API automaticamente. Confira o resultado em `docker compose logs -f api` ou na rota de saúde. O binário temporário fica no volume `api_tmp`, fora do checkout; `apps/api/tmp` também é ignorado pelo Git para quem executar o Air diretamente no host. O Air não executa as migrações SQL; use o comando `migrate` abaixo ao criar uma nova migração.

Se alterar `Dockerfile.dev`, reconstrua a imagem com `docker compose up -d --build --wait`. Alterações no `compose.yaml` ou no `.env` exigem recriar os contêineres com `docker compose up -d --force-recreate --wait`.

### cadastro de funcionários

A rota POST, o handler, o serviço e o repositório estão implementados e têm testes. O cadastro na API em execução permanece indisponível (`503`) até o login fornecer a conta do cliente. O corpo receberá somente `name`; a conta não será enviada no formulário. Não há configuração temporária de conta por variável de ambiente. A listagem GET ainda retorna `[]`.

### desenvolvimento do frontend

Abra http://localhost:5173 depois de `docker compose up -d --build --wait`. Ao salvar arquivos em `apps/web/src`, o Vite atualiza o navegador automaticamente por HMR. A página inicial verifica `/api/v1/health` pelo proxy do Vite para o Go; essa verificação aparece apenas em desenvolvimento. `WEB_PORT` altera a porta local.

As dependências do contêiner ficam em um volume separado. Ao alterar `package.json` e `pnpm-lock.yaml`, rode `docker compose restart web` para sincronizá-las com `pnpm install --frozen-lockfile`. Para build e lint dentro do Docker:

```bash
docker compose exec web pnpm run build
docker compose exec web pnpm run lint
```

## Comandos úteis

| ação | comando |
| --- | --- |
| Ver o estado dos serviços | `docker compose ps` |
| Acompanhar os logs da API | `docker compose logs -f api` |
| Consultar as migrações aplicadas | `docker compose run --rm migrate go run ./cmd/migrate status` |
| Aplicar uma migração adicionada depois da primeira subida | `docker compose run --rm migrate` |
| Parar os serviços, mantendo os dados do banco | `docker compose down` |

Para verificar o código Go fora dos contêineres, use Go 1.27.3 e execute `go test ./...` dentro de `apps/api`, onde está o `go.mod`.

## Organização

| caminho | responsabilidade |
| --- | --- |
| `apps/api/cmd/api` | Inicia a API HTTP e a conexão com o banco. |
| `apps/api/cmd/migrate` | Aplica, consulta e reverte migrações. |
| `apps/api/internal/funcionarios` | Rotas, handler de cadastro, serviço de domínio e repositório PostgreSQL. |
| `apps/api/internal/platform/postgres` | Cria e verifica o pool PostgreSQL da API. |
| `apps/api/migrations` | Migrações SQL incorporadas ao executável Go. |

`apps/web` contém a base React organizada por feature; veja a [estrutura do frontend](apps/web/README.md). O deploy unificado com a API também não foi validado. Veja [o contexto do projeto](CONTEXT.md) e as [orientações para agentes](AGENTS.md) antes de implementar novos fluxos.

## Documentação

- [bibliotecas usadas](docs/bibliotecas.md)
- [modelo de dados e relacionamentos](docs/modelo-de-dados.md)
- [requisitos](docs/requisitos.md)
- [histórias de usuário](docs/historias-de-usuario.md)
- [critérios de aceite](docs/criterios-de-aceite.md)
- [fluxos mobile](docs/fluxos-mobile.md)
- [plano de implementação](docs/plano-de-implementacao.md)
- [skills externas e suas fontes](docs/skills-externas.md)

Na documentação, **cliente** significa a pessoa que administra a produção.
