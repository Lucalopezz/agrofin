# Frontend do agrofin

Aplicação React com TypeScript, Vite, Tailwind CSS e componentes shadcn. A interface inclui uma landing page e um painel navegável de demonstração para apresentar os fluxos previstos do agrofin.

## demonstração visual

- A landing page abre em `http://localhost:5173/`; use **explorar o painel** para entrar no protótipo.
- O painel tem páginas de visão geral, colheitas, pagamentos, vendas, despesas, insumos, pulverizações, calendário e relatórios.
- Os formulários de colheita, pagamento, venda, despesa e pulverização simulam prévias, confirmações e atualizações em memória no navegador.
- Todos os dados são fictícios. A demonstração não lê nem grava lançamentos na API ou no banco e não substitui autenticação, validação ou regras de negócio do servidor.

As páginas e interações são material de apresentação; os fluxos de negócio continuam pendentes de integração e validação com o cliente.

## Desenvolvimento

Siga as instruções do [README principal](../../README.md#como-executar) para iniciar a aplicação com Docker e acesse http://localhost:5173. O Vite atualiza a página ao salvar alterações e encaminha o prefixo definido por `API_PREFIX` para a API Go. O indicador de conexão aparece somente em desenvolvimento.

```bash
docker compose logs -f web
docker compose exec web pnpm run lint
docker compose exec web pnpm run build
```

Execute os comandos na raiz do repositório. Após alterar dependências, rode `docker compose restart web` para sincronizá-las com o lockfile. Para trabalhar sem Docker, em `apps/web`, execute `pnpm install --frozen-lockfile` e `pnpm run dev`; o Vite lê `API_PREFIX` do `.env` da raiz (padrão `/api/v1`) e usa a API em `localhost:8080`. `API_PROXY_TARGET` permite mudar o destino no servidor Vite.

## Organização

- `src/app/`: composição e estilos globais.
- `src/features/`: páginas e código de cada domínio; atualmente contém `inicio`.
- `src/shared/`: componentes e utilidades compartilhados, incluindo os componentes shadcn.
- `public/images/`: marca e fotos; consulte o [catálogo de imagens](docs/imagens.md).

Mantenha requisições, tipos e componentes específicos dentro da respectiva feature. Use `shared/` para o que for compartilhado entre domínios.

## Versões

Para execução fora do Docker, use Node 24 e pnpm 12.15.0. As dependências estão em [package.json](package.json) e `pnpm-lock.yaml`.
