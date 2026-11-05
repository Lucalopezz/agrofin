# Contexto do agrofin

## Objetivo

Criar um sistema de gestão da produção de limão para um cliente que registra colheita por funcionário, pagamentos semanais, vendas ao barracão, despesas, insumos e pulverizações. O uso será prioritariamente no celular, com interface legível e lançamentos rápidos.

## Estado do projeto

Há documentação de produto, critérios de aceite, fluxos e plano técnico. O prefixo das rotas é configurado uma vez por `API_PREFIX` (padrão `/api/v1`) e compartilhado pela API Go, health check e proxy do Vite. A API oferece health check, cadastro público de clientes/gerentes, login/logout, consulta da sessão e cadastro, listagem persistida, renomeação, desativação e reativação de funcionários protegidos por sessão. `users` representa os clientes/gerentes, cada um com sua conta; `employees` representa pessoas sem login, vinculadas por `user_id`. Não há tabela `accounts`. A senha usa Argon2id; sessões opacas guardam somente o hash do token no PostgreSQL e são transportadas em cookie `HttpOnly`, `SameSite=Lax`; `Secure` é obrigatório com `PUBLIC_ORIGIN=https` e fica desativado apenas no HTTP local. As operações de escrita validam origem e exigem `X-Agrofin-Request: 1`. O frontend React + TypeScript em `apps/web`, com Tailwind e shadcn, agora tem uma landing page e um painel demonstrativo com páginas e formulários mockados; os lançamentos só atualizam o estado em memória e não usam a API. A integração do frontend e dos demais fluxos financeiros ainda não foi implementada; tarifas e a tarifa aplicada às colheitas já são protegidas na API. O deploy unificado continua sujeito à validação.

O handler de funcionários recebe somente `name` no JSON e lê `user_id` de um principal inserido pelo middleware após validar a sessão; sem sessão, as rotas retornam `401`. `POST /api/v1/auth/register` é público, recebe `name`, `login_name` e `password` e permite várias contas independentes, com nomes de acesso únicos; `login_name` não pode ser alterado pela API. `POST /api/v1/auth/login` cria uma sessão, `POST /api/v1/auth/logout` revoga a sessão e `GET /api/v1/auth/me` retorna o perfil autenticado. `PATCH /api/v1/auth/me` altera somente o nome; `PATCH /api/v1/auth/me/password` troca a senha autenticada com senha atual e nova, sem enviar confirmação do formulário, e encerra as sessões existentes; `PATCH /api/v1/auth/me/status` desativa a conta e revoga suas sessões. Uma conta desativada não pode entrar; a reativação ainda exige intervenção de operador porque não há fluxo público implementado para isso. Há testes de handler, serviço, middleware e integração PostgreSQL para auth, cadastro, listagem, renomeação, desativação e reativação de funcionários. A listagem retorna os funcionários do gerente autenticado, ordenados por nome e ID; sem registros, retorna `[]`. O filtro opcional `active=true` retorna somente ativos, `active=false` somente inativos e sua ausência preserva ambos. Valores vazios, inválidos, repetidos ou query malformada retornam `400`. A renomeação está disponível em `PATCH /api/v1/employees/{id}`, recebe somente `name` e preserva o estado ativo/inativo. O ID deve ter formato UUID; funcionário inexistente ou de outra conta retorna `404`. A desativação está disponível em `DELETE /api/v1/employees/{id}`, sem corpo: marca o registro como inativo e preenche a data de desativação, preservando o cadastro e seu histórico. Retorna `204` vazio, inclusive na repetição, sem alterar a primeira data; funcionário inexistente ou de outra conta retorna `404`. A reativação está disponível em `PATCH /api/v1/employees/{id}/reactivate`, sem corpo, com `204` vazio inclusive para registro já ativo. Restaura `active=true` e limpa o marcador `deactivated_at`, preservando ID, dono, nome e histórico; registro inexistente ou de outra conta retorna `404`. A integração do cadastro de funcionários com o frontend continua pendente.

O pacote `internal/colheitas` implementa tarifas, criação, consulta e resumo inicial de lançamentos de caixa. O pacote `internal/diarias` implementa `POST /api/v1/daily-work`, protegido por sessão, para registrar uma diária de funcionário ativo. Tarifas aplicadas e diárias usam cifra autenticada antes de serem persistidas. O chaveiro atualmente configurado pelas variáveis `HARVEST_RATE_KEYS_HEX_JSON` e `HARVEST_RATE_ACTIVE_KEY_VERSION` é compartilhado pelos dois domínios, com contextos criptográficos separados; a inicialização falha se a configuração estiver ausente ou inválida. A coleção Postman v2.1 contém uma requisição manual de diária, utilizável no Postman ou importável no Bruno e Insomnia. Pagamentos, fechamento semanal, integração com o frontend e gerenciamento operacional global de uso/rotação das chaves continuam pendentes. A regra confirmada é uma tarifa geral por tipo para todos os funcionários da mesma conta; não há tarifa individual.

## Regras centrais já descritas

- Colheita de funcionários: caixas **normais** e de **moagem**, com pagamento por caixa; diária é uma exceção.
- Tarifas de normal e moagem são gerais por conta e tipo, iguais para todos os funcionários durante a mesma vigência.
- O cadastro de funcionário pede somente o nome. O sistema gera um identificador e desativa o registro sem apagar o histórico quando necessário.
- Qualquer cliente pode criar sua conta e gerenciar somente seus dados. Funcionários são registros para colheitas e pagamentos do gerente dono, sem login. A equipe pode mudar a cada semana; participar de uma semana não exige cadastro semanal nem desativação automática na seguinte.
- O valor devido é consolidado por semana e funcionário. Dia sem colheita registrada não gera falta ou desconto.
- Venda ao barracão: caixas de **exportação**, **mercado** e **moagem**, com preços e possíveis descontos.
- Distinguir valor vendido, valor efetivamente recebido, valor devido aos funcionários e valor pago. Saldo de caixa não é lucro.
- Registrar despesas, especialmente energia da irrigação e compras de fertilizantes e defensivos.
- Registrar pulverizações com insumos cadastrados, permitir cadastro rápido e mostrar eventos e mudanças de fase da Lua no calendário.
- Proteger valores financeiros contra vazamento isolado do banco; detalhes e limites estão nos [requisitos](docs/requisitos.md).

## Direção técnica

Na API, reativação e filtro de funcionários estão implementados, assim como criação e consulta de tarifas, lançamentos e resumo inicial de colheitas por período, além do registro de diárias. O [plano de implementação](docs/plano-de-implementacao.md#funcionários-colheitas-e-agregados) e a [linguagem de domínio](docs/linguagem-de-dominio.md) detalham a separação entre cadastro, agregado de colheita e projeções de leitura: colheitas referenciam o funcionário por ID e têm seus próprios itens; o cadastro não carrega todo o histórico. Edição de colheita foi adiada para depois da V1; pagamentos, fechamento semanal, integração com o frontend, escolha do provedor operacional de chaves e controles globais de uso/rotação continuam pendentes.

A coleção HTTP versionada em `apps/api/bruno/agrofin.postman_collection.json` usa o formato Postman Collection v2.1. O Postman pode importá-la diretamente; Bruno e Insomnia também aceitam importação desse formato. As instruções do repositório priorizam Bruno e Postman; em outros clientes, confira variáveis, cookies e opções de execução após importar.

- Monorepo: `apps/api` em Go e `apps/web` em React + TypeScript.
- Backend Go com estrutura idiomática, sem impor camadas vazias. Frontend organizado por `features/<domínio>`.
- Um projeto Vercel para frontend e API, a validar com um protótipo de Vercel Services e runtime Go antes de investir na implementação completa.
- PostgreSQL adotado no desenvolvimento local; provedor de produção e estratégia final de chaves ainda não escolhidos.

## Decisões ainda abertas

O início da semana de pagamento, compras a prazo, unidades de estoque, fonte dos dados lunares e uso sem internet precisam de confirmação. A lista detalhada está em [requisitos](docs/requisitos.md#regras-e-definições-pendentes).

## Onde encontrar detalhes

- [histórias de usuário](docs/historias-de-usuario.md)
- [requisitos](docs/requisitos.md)
- [critérios de aceite](docs/criterios-de-aceite.md)
- [modelo de dados inicial](docs/modelo-de-dados.md)
- [linguagem de domínio](docs/linguagem-de-dominio.md)
- [fluxos mobile](docs/fluxos-mobile.md)
- [plano de implementação](docs/plano-de-implementacao.md)
