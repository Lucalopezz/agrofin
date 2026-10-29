# Critérios de aceite

Estes exemplos tornam as histórias verificáveis. Valores são ilustrativos; não definem tarifas reais.

## Colheita, diárias e pagamentos

### HU-01 e HU-02 — funcionários e caixas

- O cadastro pede apenas o nome. O sistema gera o identificador usado para relacionar colheitas e pagamentos, mesmo quando houver nomes iguais.
- Só o cliente entra no sistema; cadastrar um funcionário não cria usuário, senha ou convite.
- Dado um funcionário ativo e uma tarifa vigente de R$ 5,00 por caixa normal, ao registrar 10 caixas normais em um dia, o lançamento mostra R$ 50,00.
- Se a tarifa mudar para R$ 6,00 no dia seguinte, o lançamento anterior continua valendo R$ 50,00.
- Uma colheita exige data, funcionário, tipo e quantidade maior que zero. A edição registra quem alterou, quando alterou e os valores anteriores.
- Funcionários desativados deixam de aparecer na lista principal de novo lançamento, mas continuam visíveis no histórico.
- Na semana 1, lançamentos de X e Z fazem ambos aparecer no resumo. Na semana 2, lançamentos de Z e F fazem apenas Z e F aparecer no resumo daquela semana. X continua no histórico e não é desativado automaticamente.

### HU-03, HU-04 e HU-05 — fechamento semanal

- Dadas 10 caixas normais a R$ 5,00 e 4 caixas de moagem a R$ 2,00, o total por caixas é R$ 58,00.
- Se houver uma diária de R$ 80,00 na mesma semana, o valor devido fica R$ 138,00, com caixas e diária em linhas separadas.
- Se forem pagos R$ 100,00, o resumo mostra R$ 100,00 pagos e R$ 38,00 pendentes. Uma segunda tentativa de registrar o mesmo pagamento não deve gerar duplicidade.
- Um dia sem lançamento não cria falta, desconto ou alerta de ausência.
- O resumo deixa claro o intervalo de datas da semana e permite voltar ao detalhe que formou o total.

## Venda, despesas e painel

### HU-06 e HU-07 — acerto com o barracão

- Dadas 10 caixas de exportação a R$ 40,00 e 5 de mercado a R$ 30,00, o valor bruto é R$ 550,00.
- Com desconto de R$ 20,00, o valor líquido vendido é R$ 530,00; o motivo do desconto aparece no detalhe.
- Se houver recebimento de R$ 300,00, o sistema mostra R$ 230,00 a receber. Não apresenta R$ 530,00 como dinheiro já recebido.
- A quantidade por classificação da venda não é presumida a partir das caixas registradas para pagamento dos funcionários; são registros de etapas diferentes.

### HU-08, HU-09 e HU-10 — indicadores

- Um filtro de período altera todos os indicadores visíveis, com datas de início e fim explícitas.
- Energia e compras de insumos pagas aparecem como despesas com categoria e data de pagamento; o mesmo gasto não entra duas vezes por causa da movimentação de estoque.
- O painel distingue: **valor líquido vendido**, **valor recebido**, **valor devido aos funcionários** e **valor pago aos funcionários**.
- O **saldo de caixa do período** usa apenas recebimentos e saídas efetivamente pagas no período. Esse número não é apresentado como lucro contábil.
- O preço médio de venda por caixa informa se considera valor bruto ou líquido após descontos; no primeiro escopo, usar o líquido vendido dividido pelas caixas vendidas.

## Insumos e calendário

### HU-11 a HU-13 — estoque e pulverização

- Uma compra com quantidade e unidade atualiza o saldo do insumo uma única vez e registra a despesa correspondente uma única vez.
- Ao registrar pulverização, o cliente escolhe insumos existentes e informa a quantidade de cada um; o evento aparece no histórico e no calendário.
- Se o insumo não existir, o cliente o cadastra sem perder a data e os demais campos da pulverização.
- Se não souber o saldo anterior do insumo recém-cadastrado, o uso fica registrado e o estoque é sinalizado como pendente de conferência.
- O sistema guarda o que foi registrado; não recomenda doses nem garante adequação do produto à cultura.

### HU-14 — calendário lunar

- O calendário mostra o mês, as pulverizações por dia, a fase da Lua e um destaque nos dias de mudança de fase.
- Ao tocar em uma pulverização, abre o detalhe do evento. A data exibida usa o fuso horário configurado para a operação.

## Experiência e proteção

### HU-15 — uso no celular

- Os três lançamentos frequentes — caixas, pagamento e pulverização — ficam acessíveis a partir da tela inicial.
- No formulário de caixas, data atual e último funcionário podem ser sugeridos, mas são sempre editáveis antes de salvar.
- O resumo do que será salvo aparece antes da confirmação; após salvar, há retorno claro e opção de registrar outro item.
- As telas funcionam com teclado do celular aberto, sem rolagem horizontal e sem depender de hover.

### HU-16 — confidencialidade financeira

- Uma cópia isolada do banco não revela tarifas, preços, pagamentos, recebimentos ou despesas em texto claro.
- A aplicação só devolve valores financeiros a uma sessão autorizada; logs, erros e telemetria não registram esses valores.
- A restauração de backup recupera dados cifrados e as chaves necessárias, sem colocar as chaves dentro do mesmo backup do banco.
