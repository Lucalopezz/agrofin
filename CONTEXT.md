# contexto do agrofin

## objetivo

Criar um sistema de gestão da produção de limão para um cliente que registra colheita por funcionário, pagamentos semanais, vendas ao barracão, despesas, insumos e pulverizações. O uso será prioritariamente no celular, com interface legível e lançamentos rápidos.

## estado do projeto

Há documentação de produto, critérios de aceite, fluxos e plano técnico. A API Go em `apps/api` oferece `GET /api/v1/health`, uma listagem vazia em `GET /api/v1/employees` e `POST /api/v1/employees`, indisponível (`503`) até a integração com o login. O PostgreSQL local, as migrações do esquema e a conexão inicial da API estão implementados. Ainda não há frontend React, autenticação, criptografia de valores, fluxos de negócio completos ou deploy validado. O deploy unificado continua sujeito à validação.

O handler recebe somente `name` no JSON, chama `Service.Create` com `r.Context()` e uma conta definida no servidor e devolve `201` com ID, nome e estado ativo. O serviço valida e normaliza o nome, monta um `Employee` e entrega essa entidade a `Repository.Create`, que insere em `employees` com UUID gerado pelo banco. O `main` compõe o fluxo. A integração com o login ainda não fornece a conta do cliente; por isso o `main` monta o handler sem conta e o POST retorna `503`. Não há configuração temporária de conta por variável de ambiente. Há testes de handler, serviço e integração HTTP com PostgreSQL. A origem autenticada de `account_id`, a listagem persistida, a renomeação e a desativação continuam pendentes.

## regras centrais já descritas

- Colheita de funcionários: caixas **normais** e de **moagem**, com pagamento por caixa; diária é uma exceção.
- O cadastro de funcionário pede somente o nome. O sistema gera um identificador e desativa o registro sem apagar o histórico quando necessário.
- Só o cliente, gerente da plantação, acessa o sistema. Funcionários são registros para colheitas e pagamentos, sem login. A equipe pode mudar a cada semana; participar de uma semana não exige cadastro semanal nem desativação automática na seguinte.
- O valor devido é consolidado por semana e funcionário. Dia sem colheita registrada não gera falta ou desconto.
- Venda ao barracão: caixas de **exportação**, **mercado** e **moagem**, com preços e possíveis descontos.
- Distinguir valor vendido, valor efetivamente recebido, valor devido aos funcionários e valor pago. Saldo de caixa não é lucro.
- Registrar despesas, especialmente energia da irrigação e compras de fertilizantes e defensivos.
- Registrar pulverizações com insumos cadastrados, permitir cadastro rápido e mostrar eventos e mudanças de fase da Lua no calendário.
- Proteger valores financeiros contra vazamento isolado do banco; detalhes e limites estão nos [requisitos](docs/requisitos.md).

## direção técnica

- Monorepo: `apps/api` em Go e `apps/web` em React + TypeScript.
- Backend Go com estrutura idiomática, sem impor camadas vazias. Frontend organizado por `features/<domínio>`.
- Um projeto Vercel para frontend e API, a validar com um protótipo de Vercel Services e runtime Go antes de investir na implementação completa.
- PostgreSQL adotado no desenvolvimento local; provedor de produção e estratégia final de chaves ainda não escolhidos.

## decisões ainda abertas

O início da semana de pagamento, possível tarifa individual, compras a prazo, unidades de estoque, fonte dos dados lunares e uso sem internet precisam de confirmação. A lista detalhada está em [requisitos](docs/requisitos.md#regras-e-definições-pendentes).

## onde encontrar detalhes

- [histórias de usuário](docs/historias-de-usuario.md)
- [requisitos](docs/requisitos.md)
- [critérios de aceite](docs/criterios-de-aceite.md)
- [modelo de dados inicial](docs/modelo-de-dados.md)
- [fluxos mobile](docs/fluxos-mobile.md)
- [plano de implementação](docs/plano-de-implementacao.md)
