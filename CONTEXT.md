# Contexto do agrofin

## Objetivo

Criar um sistema de gestão da produção de limão para um cliente que registra colheita por funcionário, pagamentos semanais, vendas ao barracão, despesas, insumos e pulverizações. O uso será prioritariamente no celular, com interface legível e lançamentos rápidos.

## Estado do projeto

Há documentação de produto, critérios de aceite, fluxos e plano técnico. O prefixo das rotas é configurado uma vez por `API_PREFIX` (padrão `/api/v1`) e compartilhado pela API Go, health check e proxy do Vite. A API oferece health check, cadastro inicial do gerente, login/logout, consulta da sessão e rotas de funcionários protegidas por sessão; a listagem continua vazia. `users` representa o gerente; `employees` representa pessoas sem login, vinculadas por `user_id`. Não há tabela `accounts`. A senha usa Argon2id; sessões opacas guardam somente o hash do token no PostgreSQL e são transportadas em cookie `HttpOnly`, `SameSite=Lax`; `Secure` é obrigatório com `PUBLIC_ORIGIN=https` e fica desativado apenas no HTTP local. As operações de escrita validam origem e exigem `X-Agrofin-Request: 1`. Há uma base React + TypeScript em `apps/web`, com Tailwind, shadcn, tema claro e Vite com HMR no Docker; a página inicial verifica a conexão com o Go pelo proxy. Ainda não há telas de negócio, criptografia de valores, fluxos de negócio completos ou deploy validado. O deploy unificado continua sujeito à validação.

O handler de funcionários recebe somente `name` no JSON e lê `user_id` de um principal inserido pelo middleware após validar a sessão; sem sessão, as rotas retornam `401`. `POST /api/v1/auth/register` cria o único gerente, exige `X-Agrofin-Setup-Token` e recebe `login_name` e `password`; a senha nunca aparece em logs. `POST /api/v1/auth/login` cria uma sessão, `POST /api/v1/auth/logout` revoga a sessão e `GET /api/v1/auth/me` confirma uma sessão válida. Há testes de handler, serviço, middleware e integração PostgreSQL para sessões e cadastro de funcionários. A listagem persistida, renomeação e desativação de funcionários continuam pendentes.

## Regras centrais já descritas

- Colheita de funcionários: caixas **normais** e de **moagem**, com pagamento por caixa; diária é uma exceção.
- O cadastro de funcionário pede somente o nome. O sistema gera um identificador e desativa o registro sem apagar o histórico quando necessário.
- Só o cliente, gerente da plantação, acessa o sistema. Funcionários são registros para colheitas e pagamentos, sem login. A equipe pode mudar a cada semana; participar de uma semana não exige cadastro semanal nem desativação automática na seguinte.
- O valor devido é consolidado por semana e funcionário. Dia sem colheita registrada não gera falta ou desconto.
- Venda ao barracão: caixas de **exportação**, **mercado** e **moagem**, com preços e possíveis descontos.
- Distinguir valor vendido, valor efetivamente recebido, valor devido aos funcionários e valor pago. Saldo de caixa não é lucro.
- Registrar despesas, especialmente energia da irrigação e compras de fertilizantes e defensivos.
- Registrar pulverizações com insumos cadastrados, permitir cadastro rápido e mostrar eventos e mudanças de fase da Lua no calendário.
- Proteger valores financeiros contra vazamento isolado do banco; detalhes e limites estão nos [requisitos](docs/requisitos.md).

## Direção técnica

- Monorepo: `apps/api` em Go e `apps/web` em React + TypeScript.
- Backend Go com estrutura idiomática, sem impor camadas vazias. Frontend organizado por `features/<domínio>`.
- Um projeto Vercel para frontend e API, a validar com um protótipo de Vercel Services e runtime Go antes de investir na implementação completa.
- PostgreSQL adotado no desenvolvimento local; provedor de produção e estratégia final de chaves ainda não escolhidos.

## Decisões ainda abertas

O início da semana de pagamento, possível tarifa individual, compras a prazo, unidades de estoque, fonte dos dados lunares e uso sem internet precisam de confirmação. A lista detalhada está em [requisitos](docs/requisitos.md#regras-e-definições-pendentes).

## Onde encontrar detalhes

- [histórias de usuário](docs/historias-de-usuario.md)
- [requisitos](docs/requisitos.md)
- [critérios de aceite](docs/criterios-de-aceite.md)
- [modelo de dados inicial](docs/modelo-de-dados.md)
- [fluxos mobile](docs/fluxos-mobile.md)
- [plano de implementação](docs/plano-de-implementacao.md)
