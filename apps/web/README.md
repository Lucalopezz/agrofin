# Frontend do agrofin

Aplicação React com TypeScript, Vite, Tailwind CSS e componentes shadcn. A página inicial está disponível; as telas de negócio ainda serão implementadas.

## Desenvolvimento

Siga as instruções do [README principal](../../README.md#como-executar) para iniciar a aplicação com Docker e acesse http://localhost:5173. O Vite atualiza a página ao salvar alterações e encaminha `/api` para a API Go. O indicador de conexão aparece somente em desenvolvimento.

```bash
docker compose logs -f web
docker compose exec web pnpm run lint
docker compose exec web pnpm run build
```

Execute os comandos na raiz do repositório. Após alterar dependências, rode `docker compose restart web` para sincronizá-las com o lockfile. Para trabalhar sem Docker, em `apps/web`, execute `pnpm install --frozen-lockfile` e `pnpm run dev`; o proxy usa a API em `localhost:8080` por padrão. `API_PROXY_TARGET` permite mudar o destino no servidor Vite.

## Organização

- `src/app/`: composição e estilos globais.
- `src/features/`: páginas e código de cada domínio; atualmente contém `inicio`.
- `src/shared/`: componentes e utilidades compartilhados, incluindo os componentes shadcn.
- `public/images/`: marca e fotos; consulte o [catálogo de imagens](docs/imagens.md).

Mantenha requisições, tipos e componentes específicos dentro da respectiva feature. Use `shared/` para o que for compartilhado entre domínios.

## Versões

Para execução fora do Docker, use Node 24 e pnpm 12.15.0. As dependências estão em [package.json](package.json) e `pnpm-lock.yaml`.
