# Fluxos mobile

As telas abaixo descrevem o caminho mais curto para as tarefas frequentes. São hipóteses para testar com o cliente, não um desenho visual fechado.

## Início

- Mostrar três ações grandes: **registrar caixas**, **registrar pagamento** e **registrar pulverização**.
- Mostrar um resumo curto da semana: caixas, valor devido aos funcionários e pagamentos pendentes.
- Deixar **vendas**, **despesas**, **insumos**, **calendário** e **relatórios** em navegação clara, sempre acessível.

## Registrar caixas

1. Abrir a ação direta da tela inicial.
2. Confirmar ou trocar a data; selecionar funcionário.
3. Informar quantidade de caixas normais e/ou de moagem no mesmo formulário. Campos vazios significam zero; pelo menos um tipo precisa ter quantidade positiva.
4. Conferir o total calculado e salvar.
5. Oferecer **registrar outro funcionário** mantendo a data, ou **ver resumo do dia**.

Uma ação de **repetir lançamento** pode preencher funcionário e tipos do último registro, mas nunca deve salvar sem revisão. Se a tarifa usada for diferente da esperada, a tela mostra o valor unitário antes da confirmação.

## Fechar a semana e pagar

1. Abrir **pagamentos** e escolher a semana, já sugerida como a atual ou a última com saldo.
2. Ver cada funcionário com caixas por tipo, diárias, total devido, total pago e saldo.
3. Tocar em um funcionário, conferir o detalhamento e informar valor e data do pagamento.
4. Salvar e voltar à lista, que já mostra o saldo atualizado.

Correções de caixas ou diárias de uma semana quitada exigem destaque do novo saldo e um registro de ajuste, para que o histórico continue compreensível.

## Registrar venda

1. Informar data e barracão.
2. Preencher quantidade e preço de exportação, mercado e moagem apenas para as classificações presentes.
3. Incluir descontos com motivo, se houver.
4. Conferir bruto, desconto e líquido; salvar.
5. Registrar recebimento na hora ou depois. A tela mostra quanto ainda falta receber.

## Registrar pulverização

1. Abrir a ação direta e confirmar a data.
2. Escolher os insumos usados e suas quantidades; incluir observação opcional.
3. Se faltar um insumo, abrir um cadastro curto sobre o formulário atual. Ao voltar, o novo insumo já está selecionado.
4. Conferir o registro e salvar. Se o estoque for desconhecido, mostrar aviso de conferência pendente sem perder o evento.
5. Abrir o evento salvo no calendário, se desejado.

## Critérios de interface a observar em campo

- O cliente entende os nomes usados para tipos de caixa, barracão e insumos sem explicação externa.
- Ele consegue registrar caixas de vários funcionários em sequência sem redigitar a data.
- Os números principais e o botão de salvar ficam visíveis com o teclado do celular aberto.
- É possível corrigir um erro recente sem apagar o histórico.
- As fases da Lua são visualmente distinguíveis e os dias de mudança aparecem por escrito, além de ícone ou cor.
