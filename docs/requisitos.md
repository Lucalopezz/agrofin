# requisitos

## escopo inicial

### cadastro e colheita

- **RF-01** Cadastrar e renomear funcionários informando apenas o nome. O sistema gera o identificador e controla o estado ativo/inativo; a exclusão na interface desativa o registro e preserva seu histórico.
- **RF-02** Configurar valores por caixa normal e de moagem, com vigência por data. Alterações futuras não recalculam pagamentos anteriores.
- **RF-03** Registrar colheita por data, funcionário, tipo de caixa e quantidade inteira positiva; permitir correção com histórico de alteração. O valor unitário aplicado deve ficar gravado no lançamento.
- **RF-04** Consolidar colheita por semana e funcionário, mostrando quantidades, tarifa aplicada e total devido.
- **RF-05** Registrar pagamento parcial ou integral, data e observação; mostrar saldo pendente sem duplicar pagamentos. Não permitir pagar além do devido sem um ajuste explícito.
- **RF-06** Registrar diária como lançamento excepcional, com funcionário, data e valor; sua contabilização é separada das caixas.
- **RF-07** Não criar faltas, penalidades ou descontos automáticos quando não houver registro de colheita em um dia.

### vendas e despesas

- **RF-08** Registrar venda ao barracão por data, com linhas de exportação, mercado e moagem; cada linha contém quantidade e preço unitário.
- **RF-09** Registrar desconto da venda com valor e motivo; mostrar total bruto, desconto e total líquido. Um desconto não pode exceder o valor bruto da venda.
- **RF-10** Registrar recebimentos da venda, inclusive parciais, e distinguir valor vendido de valor recebido.
- **RF-11** Registrar despesas por categoria, data de pagamento, valor e descrição; incluir energia de irrigação, fertilizantes, defensivos e outras despesas. Uma compra registrada como despesa não deve gerar outra saída de caixa quando o insumo for usado.
- **RF-12** Exibir indicadores por período: caixas colhidas por tipo e funcionário, caixas vendidas por classificação, preço médio de venda por caixa, remuneração de funcionários, descontos, despesas, valor vendido, valor recebido e saldo de caixa. Não chamar o saldo de caixa de lucro.

### insumos e pulverização

- **RF-13** Cadastrar insumos com nome, categoria (fertilizante ou defensivo), unidade e observações.
- **RF-14** Registrar compras de insumos com quantidade, unidade, custo e data; atualizar o estoque.
- **RF-15** Registrar pulverização com data, insumos e quantidades usadas, observações e, se necessário, cadastro rápido de novo insumo no mesmo fluxo.
- **RF-16** Dar baixa no estoque conforme o uso registrado. No cadastro rápido, permitir informar um saldo inicial ou marcar o estoque como pendente de conferência; não inventar uma compra para justificar o uso.
- **RF-17** Mostrar calendário mensal com pulverizações, fase da Lua em cada dia e datas de transição entre fases, usando o fuso horário local.

## requisitos não funcionais

- **RNF-01 — mobile primeiro:** os fluxos de colheita, pagamento e pulverização devem funcionar integralmente em uma tela estreita, com campos e botões grandes e texto legível.
- **RNF-02 — poucos passos:** permitir repetir um lançamento recente, usar valores padrão editáveis e concluir os lançamentos frequentes em uma única tela sempre que possível.
- **RNF-03 — clareza:** mostrar prévia do total antes de salvar, confirmação após salvar, erros em linguagem direta e identificação explícita de valores pendentes, pagos, vendidos e recebidos.
- **RNF-04 — acesso:** autenticar o cliente; começar com uma conta administradora e deixar a ampliação de perfis para decisão posterior. Funcionário cadastrado não precisa de login no primeiro escopo.
- **RNF-05 — segurança:** usar HTTPS, controle de acesso no servidor, validação de entradas, backups protegidos e exclusão de dados financeiros dos logs.
- **RNF-06 — criptografia:** cifrar valores financeiros sensíveis na aplicação antes de persistir, com criptografia autenticada e chaves gerenciadas fora do banco; prever rotação e recuperação das chaves. Senhas devem usar hash próprio para senhas. A proteção cobre vazamento isolado do banco, não comprometimento simultâneo da aplicação e das chaves.
- **RNF-07 — integridade:** operações de colheita, pagamentos, vendas e estoque devem ser transacionais; correções precisam deixar rastreabilidade.
- **RNF-08 — desempenho:** telas de lançamento devem responder bem em conexão móvel comum; indicadores podem ser calculados no servidor por período.
- **RNF-09 — recuperação:** manter backups restauráveis e testar periodicamente a restauração, inclusive das chaves necessárias para ler os valores cifrados.

## regras e definições pendentes

- Definir se a semana de pagamento começa na segunda-feira ou em outro dia.
- Confirmar se as tarifas por caixa variam por funcionário, por período ou por tipo de serviço.
- Confirmar se a classificação da venda é informada diretamente pelo barracão e se uma venda pode reunir vários tipos de caixa.
- Definir se o estoque admite unidades diferentes e conversões (por exemplo, litro e mililitro).
- Definir a origem dos dados das fases da Lua e se basta exibir a fase ou também o horário exato da mudança.
- Confirmar se haverá uso sem internet e se outras pessoas terão acesso ao sistema.
- Confirmar se os pagamentos e recebimentos parciais acontecem na prática; o desenho inicial os admite para evitar perda de informação.
- Confirmar como lançar correções feitas após uma semana já paga ou uma venda já recebida.
- Confirmar se compras de insumos podem ser feitas a prazo. O desenho inicial de saldo de caixa considera despesas registradas como já pagas.

## fora do primeiro escopo

- Folha de pagamento formal, controle de ponto e penalização por ausência.
- Recomendação de produto, dose ou momento de aplicação de defensivos.
- Integração automática com barracão, banco, medidor de energia ou notas fiscais.
