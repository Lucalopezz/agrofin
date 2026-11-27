# Frontend do agrofin

Aplicação React com TypeScript, Vite, Tailwind CSS e componentes shadcn. A landing page oferece cadastro e login. O painel usa o cookie de sessão para acessar a API.

## Fluxos do painel

- Funcionários, tarifas, colheitas, pagamentos, vendas, despesas e relatórios estão integrados à API. As páginas de insumos, pulverizações e calendário são demonstrativas.
- Em **Pagamentos**, a tela abre na semana atual. Aplique outro período quando precisar conferir as caixas, as diárias, o devido, o pago e o saldo por funcionário.
- Use **Registrar diária** para lançar o trabalho de um funcionário ativo. Use **Ver lançamentos** para conferir as caixas e as diárias que compõem a folha.
- Use **Pagar** para registrar o valor integral ou parcial no intervalo exibido. A API informa a data efetiva do pagamento e atualiza o saldo da folha.
- Em **Vendas**, registre as caixas e preços por classificação e informe o desconto com motivo quando houver. Após salvar, abra o recebimento na venda para registrar a entrada parcial ou integral. A lista filtra pela data da venda e mostra o saldo atualizado de cada acerto.

## Desenvolvimento

Siga as instruções do [README principal](../../README.md#como-executar) para iniciar a aplicação com Docker e acesse http://localhost:5173. O Vite atualiza a página ao salvar alterações e encaminha o prefixo definido por `API_PREFIX` para a API Go. O indicador de conexão aparece somente em desenvolvimento.

```bash
docker compose logs -f web
docker compose exec web pnpm run test
docker compose exec web pnpm run lint
docker compose exec web pnpm run build
```

Execute os comandos na raiz do repositório. Após alterar dependências, rode `docker compose restart web` para sincronizá-las com o lockfile. Para trabalhar sem Docker, em `apps/web`, execute `pnpm install --frozen-lockfile` e `pnpm run dev`; o Vite lê `API_PREFIX` do `.env` da raiz (padrão `/api/v1`) e usa a API em `localhost:8080`. `API_PROXY_TARGET` permite mudar o destino no servidor Vite.

## Organização

- `src/app/` reúne a composição e os estilos globais.
- `src/features/` organiza as páginas e o código por domínio.
- `src/shared/` reúne componentes e utilidades compartilhados, incluindo os componentes shadcn.
- `public/images/` reúne a marca e as fotos. [Catálogo de imagens](docs/imagens.md)

Mantenha requisições, tipos e componentes específicos dentro da respectiva feature. Use `shared/` para o que for compartilhado entre domínios.

## Versões

Para execução fora do Docker, use Node 24 e pnpm 12.15.0. As dependências estão em [package.json](package.json) e `pnpm-lock.yaml`.
