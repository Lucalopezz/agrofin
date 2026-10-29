# frontend do agrofin

Base React + TypeScript com Vite, Tailwind CSS e componentes shadcn. Gerenciador: pnpm 12.15.0 (fixado em `packageManager`). Interface em português, tema claro e layout pensado para celular. Esta entrega contém somente a estrutura e uma página inicial; os fluxos de negócio e o deploy continuam pendentes.

## estrutura

```text
src/
  app/
    app.tsx                 # composição da aplicação; futuras rotas ficam aqui
    styles.css              # estilos base e tokens globais do tema
  features/
    inicio/
      pages/home-page.tsx
      components/api-status.tsx
  shared/
    components/ui/button.tsx # componente shadcn
    lib/utils.ts             # composição de classes
```

Cada novo domínio pertence a `features/<domínio>`. Crie `components/`, `pages/`, `hooks/`, `api/` e arquivos de tipos dentro da feature conforme houver uso real. Requisições e tipos específicos não vão para `shared/`. Componentes globais e utilidades independentes do domínio ficam em `shared/`; não criar pastas ou abstrações vazias. Rotas e provedores globais serão compostos em `app/` quando necessários.

`components.json` configura os aliases do shadcn para `shared/`. Os componentes são código local editável. O botão inicial veio do registro oficial, com importação de `cn` adaptada e área de toque padrão de 44 px. Ao adicionar componentes, confira se o gerador preservou o alias `@/shared/lib/utils`.

## desenvolvimento

Na raiz do repositório, configure `.env` conforme o README e execute:

```bash
docker compose up -d --build --wait
```

Abra http://localhost:5173. O bind mount entrega os arquivos ao Vite, que aplica HMR sem reiniciar o contêiner. O proxy encaminha `/api` para `http://api:8080`; não há URL interna de Docker no bundle nem configuração de CORS necessária. O indicador de conexão é exclusivo de desenvolvimento, trata falha e permite tentar novamente. Não representa disponibilidade dos fluxos de negócio ou verificação contínua do banco.

```bash
docker compose logs -f web
docker compose exec web pnpm run build
docker compose exec web pnpm run lint
```

O volume `web_pnpm_modules` separa dependências do host e do contêiner; `pnpm install --frozen-lockfile` sincroniza o lockfile em cada início. Depois de mudar dependências, reinicie `web`. Alterações no Dockerfile exigem rebuild. Para executar sem Docker, use Node 24, `pnpm install --frozen-lockfile` e `pnpm run dev`; o proxy assume a API em `localhost:8080`. `API_PROXY_TARGET` é uma configuração do servidor Vite, sem prefixo `VITE_`, para não expor infraestrutura ao navegador.

## paleta clara

| token | cor | uso |
| --- | --- | --- |
| background | `#f7faf7` | fundo levemente verde |
| card | `#ffffff` | superfícies brancas |
| primary | `#24633f` | ações e destaques |
| foreground | `#183326` | texto principal |
| muted-foreground | `#526557` | texto secundário |
| secondary | `#eaf3e7` | superfícies suaves |
| accent | `#e3efce` | subtom de limão |

Os tokens semânticos em `app/styles.css` alimentam Tailwind e shadcn. A fonte usa a pilha do sistema, evitando download externo. Não há tema escuro nesta etapa.
