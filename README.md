# Agrofin

O agrofin é uma aplicação para ajudar o cliente a administrar a produção de limão pelo celular. A proposta é reunir registros de colheita por funcionário, pagamentos, vendas, despesas, insumos e pulverizações em uma interface simples e rápida de usar.

## Estado do projeto

O projeto está em desenvolvimento. Existe uma página inicial e uma API conectada ao PostgreSQL, com migrações do banco. A rota de saúde funciona; a listagem de funcionários ainda retorna uma lista vazia e o cadastro responde `503` até a integração com o login. Os demais fluxos, a proteção dos valores financeiros e o deploy ainda não estão prontos.

## Tecnologias

- **Frontend:** React, TypeScript, Vite, Tailwind CSS e componentes shadcn.
- **Backend:** Go, `net/http`, pgx e goose para migrações SQL.
- **Banco:** PostgreSQL.
- **Desenvolvimento:** Docker Compose, pnpm e Air para recarga da API.

## Estrutura

```text
apps/
  api/                 API Go, testes e migrações SQL
  web/                 aplicação React
    src/app/           composição e estilos globais
    src/features/      telas e código por domínio
    src/shared/        componentes e utilidades compartilhados
docs/                  requisitos, fluxos e decisões do projeto
compose.yaml           serviços para desenvolvimento local
```

## Como executar

É necessário ter Git e Docker com Compose instalados.

1. Clone o repositório e entre na pasta:

   ```bash
   git clone https://github.com/Lucalopezz/agrofin.git
   cd agrofin
   ```

2. Crie a configuração local:

   ```bash
   cp .env.example .env
   ```

   No `.env`, substitua `POSTGRES_PASSWORD` por uma senha hexadecimal longa. Se já tiver um `.env`, preserve as credenciais existentes.

3. Inicie os serviços:

   ```bash
   docker compose up -d --build --wait
   ```

   O Compose inicia o banco, aplica as migrações e sobe a API e o frontend. Acesse [localhost:5173](http://localhost:5173). A API está em `http://127.0.0.1:8080`; confira sua resposta com:

   ```bash
   curl http://127.0.0.1:8080/api/v1/health
   ```

As portas podem ser alteradas no `.env`. Alterações no código atualizam o frontend e reiniciam a API automaticamente.

## Comandos úteis

| ação | comando |
| --- | --- |
| Ver serviços | `docker compose ps` |
| Acompanhar logs | `docker compose logs -f api web` |
| Aplicar novas migrações | `docker compose run --rm migrate` |
| Consultar migrações | `docker compose run --rm migrate go run ./cmd/migrate status` |
| Verificar o frontend | `docker compose exec web pnpm run lint` e `docker compose exec web pnpm run build` |
| Parar serviços, preservando o banco | `docker compose down` |

Para executar os testes Go, rode `go test ./...` em `apps/api`. O CI também verifica migrações, integração com PostgreSQL e a stack Docker. Após alterar dependências do frontend, reinicie o serviço com `docker compose restart web`; mudanças nos Dockerfiles exigem `docker compose up -d --build --wait`.

## Documentação

- [Requisitos e decisões pendentes](docs/requisitos.md)
- [Histórias de usuário](docs/historias-de-usuario.md) e [critérios de aceite](docs/criterios-de-aceite.md)
- [Fluxos para celular](docs/fluxos-mobile.md)
- [Modelo de dados](docs/modelo-de-dados.md) e [plano de implementação](docs/plano-de-implementacao.md)
- [Frontend](apps/web/README.md) e [bibliotecas](docs/bibliotecas.md)
- [Contexto atual](CONTEXT.md) e [orientações para agentes](AGENTS.md)

Os documentos de produto descrevem também funcionalidades planejadas; consulte o estado do projeto acima para distinguir o que já existe.

## Versões

O ambiente Docker usa Go 1.27.3 e PostgreSQL 17. Para desenvolvimento do frontend fora do Docker, use Node 24 e pnpm 12.15.0. As versões das dependências estão em [go.mod](apps/api/go.mod), [package.json](apps/web/package.json) e nos respectivos arquivos de lock.
