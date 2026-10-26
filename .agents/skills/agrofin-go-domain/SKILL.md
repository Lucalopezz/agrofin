---
name: agrofin-go-domain
description: Implementar ou revisar regras da colheita, pagamentos, vendas, despesas, insumos e pulverizações na API Go do agrofin.
---

# domínio em go

Leia [o contexto](../../../CONTEXT.md), os [requisitos](../../../docs/requisitos.md) e os [critérios de aceite](../../../docs/criterios-de-aceite.md) do fluxo afetado. Confira o código existente antes de tomar o plano como implementação.

## preservar regras do negócio

- Colheita de funcionário usa normal/moagem; venda ao barracão usa exportação/mercado/moagem. Não converta uma classificação na outra sem dado explícito.
- Fotografe a tarifa aplicada a cada colheita para que uma mudança posterior não altere o passado. Separe diária de quantidade de caixas.
- Semana e funcionário delimitam o valor devido. Um dia sem lançamento não é ausência penalizável. Pagamentos parciais reduzem o saldo, e tentativas duplicadas não devem criar nova saída.
- Desconto de venda altera o valor líquido vendido; recebimento é um evento separado. Compra de insumo afeta despesa e estoque uma única vez; pulverização afeta uso de estoque, sem repetir a despesa.
- Para uma correção após pagamento ou recebimento, preserve rastreabilidade e mostre o novo saldo. Consulte as decisões pendentes quando o comportamento ainda não estiver definido.

## organizar a implementação

- Use pacotes Go com responsabilidade clara em `internal/`; `cmd/` hospeda a entrada local quando necessário. Mantenha HTTP, validação de entrada e persistência fora das funções que calculam regras de domínio.
- Faça cálculos monetários com centavos inteiros e arredondamento definido. Deixe operações que mudam registros relacionados dentro de transações.
- Trate o servidor como fonte dos totais. Evite abstrações e camadas que ainda não resolvem uma necessidade concreta.
- Ao alterar uma regra, verifique pelo menos o caso principal, um limite e a persistência quando ela participar da invariância. Explique brevemente o motivo da organização escolhida para facilitar o estudo de Go.
