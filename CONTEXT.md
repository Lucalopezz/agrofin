# contexto do agrofin

## objetivo

Criar um sistema de gestão da produção de limão para um cliente que registra colheita por funcionário, pagamentos semanais, vendas ao barracão, despesas, insumos e pulverizações. O uso será prioritariamente no celular, com interface legível e lançamentos rápidos.

## estado do projeto

Há documentação de produto, critérios de aceite, fluxos e plano técnico. Ainda não há código da aplicação. A estrutura Go/React e o deploy unificado são decisões de direção, sujeitas à validação do primeiro protótipo.

## regras centrais já descritas

- Colheita de funcionários: caixas **normais** e de **moagem**, com pagamento por caixa; diária é uma exceção.
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
- Banco relacional previsto; provedor e estratégia final de chaves ainda não escolhidos.

## decisões ainda abertas

O início da semana de pagamento, possível tarifa individual, compras a prazo, unidades de estoque, fonte dos dados lunares, uso sem internet e acesso por outras pessoas precisam de confirmação. A lista detalhada está em [requisitos](docs/requisitos.md#regras-e-definições-pendentes).

## onde encontrar detalhes

- [histórias de usuário](docs/historias-de-usuario.md)
- [requisitos](docs/requisitos.md)
- [critérios de aceite](docs/criterios-de-aceite.md)
- [modelo de dados inicial](docs/modelo-de-dados.md)
- [fluxos mobile](docs/fluxos-mobile.md)
- [plano de implementação](docs/plano-de-implementacao.md)
