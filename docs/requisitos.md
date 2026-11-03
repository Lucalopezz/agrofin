# Requisitos

## Escopo inicial

### Cadastro e colheita

- **RF-01** Cadastrar e renomear funcionários informando apenas o nome, sem criar acesso ao sistema para eles. O sistema gera o identificador e controla o estado ativo/inativo; a exclusão na interface desativa o registro e preserva seu histórico. Permitir reativar o mesmo cadastro e filtrar a listagem por ativos ou inativos. A equipe de uma semana é formada pelos funcionários com lançamentos naquele período, sem cadastro semanal ou desativação automática dos demais. A reativação e o filtro estão implementados na API, conforme o plano de implementação.
- **RF-02** Configurar tarifas gerais por conta e tipo de caixa normal ou moagem, iguais para todos os funcionários da conta na mesma vigência, em centavos inteiros maiores que zero. Não há exceções individuais. O início é obrigatório e inclusivo; o fim é opcional e exclusivo, sempre posterior ao início quando informado. Alterações futuras não recalculam pagamentos anteriores. Domínio, serviço, proteção, repositório PostgreSQL e `POST /harvest-rates` estão implementados. A listagem de tarifas e os controles operacionais globais de uso e rotação de chaves continuam pendentes.
- **RF-03** Registrar colheita por data, funcionário, tipo de caixa e quantidade inteira positiva; permitir correção com histórico de alteração. O valor unitário aplicado deve ficar gravado no lançamento.
- **RF-04** Consolidar colheita por semana e funcionário, mostrando quantidades, tarifa aplicada e total devido.
- **RF-05** Registrar pagamento parcial ou integral, data e observação; mostrar saldo pendente sem duplicar pagamentos. Não permitir pagar além do devido sem um ajuste explícito.
- **RF-06** Registrar diária como lançamento excepcional, com funcionário, data e valor; sua contabilização é separada das caixas.
- **RF-07** Não criar faltas, penalidades ou descontos automáticos quando não houver registro de colheita em um dia.

### Vendas e despesas

- **RF-08** Registrar venda ao barracão por data, com linhas de exportação, mercado e moagem; cada linha contém quantidade e preço unitário.
- **RF-09** Registrar desconto da venda com valor e motivo; mostrar total bruto, desconto e total líquido. Um desconto não pode exceder o valor bruto da venda.
- **RF-10** Registrar recebimentos da venda, inclusive parciais, e distinguir valor vendido de valor recebido.
- **RF-11** Registrar despesas por categoria, data de pagamento, valor e descrição; incluir energia de irrigação, fertilizantes, defensivos e outras despesas. Uma compra registrada como despesa não deve gerar outra saída de caixa quando o insumo for usado.
- **RF-12** Exibir indicadores por período: caixas colhidas por tipo e funcionário, caixas vendidas por classificação, preço médio de venda por caixa, remuneração de funcionários, descontos, despesas, valor vendido, valor recebido e saldo de caixa. Não chamar o saldo de caixa de lucro.

### Insumos e pulverização

- **RF-13** Cadastrar insumos com nome, categoria (fertilizante ou defensivo), unidade e observações.
- **RF-14** Registrar compras de insumos com quantidade, unidade, custo e data; atualizar o estoque.
- **RF-15** Registrar pulverização com data, insumos e quantidades usadas, observações e, se necessário, cadastro rápido de novo insumo no mesmo fluxo.
- **RF-16** Dar baixa no estoque conforme o uso registrado. No cadastro rápido, permitir informar um saldo inicial ou marcar o estoque como pendente de conferência; não inventar uma compra para justificar o uso.
- **RF-17** Mostrar calendário mensal com pulverizações, fase da Lua em cada dia e datas de transição entre fases, usando o fuso horário local.

### Contas e acesso

- **RF-18** Permitir que qualquer cliente crie sua conta com nome, nome de acesso e senha, sem convite ou token de instalação. O nome de acesso é único; contas distintas mantêm funcionários e lançamentos separados por `user_id`.

## Requisitos não funcionais

- **RNF-01 — mobile primeiro:** os fluxos de colheita, pagamento e pulverização devem funcionar integralmente em uma tela estreita, com campos e botões grandes e texto legível.
- **RNF-02 — poucos passos:** permitir repetir um lançamento recente, usar valores padrão editáveis e concluir os lançamentos frequentes em uma única tela sempre que possível.
- **RNF-03 — clareza:** mostrar prévia do total antes de salvar, confirmação após salvar, erros em linguagem direta e identificação explícita de valores pendentes, pagos, vendidos e recebidos.
- **RNF-04 — acesso:** permitir cadastro público e autenticação de vários clientes, cada um gerente da própria produção. Cada usuário acessa somente os próprios funcionários e lançamentos. Funcionários cadastrados não possuem login nem acesso às telas.
- **RNF-05 — segurança:** usar HTTPS, controle de acesso no servidor, validação de entradas, backups protegidos e exclusão de dados financeiros dos logs.
- **RNF-06 — criptografia:** cifrar valores financeiros sensíveis na aplicação antes de persistir, com criptografia autenticada e chaves gerenciadas fora do banco; prever rotação e recuperação das chaves. Senhas devem usar hash próprio para senhas. A proteção cobre vazamento isolado do banco, não comprometimento simultâneo da aplicação e das chaves.
- **RNF-07 — integridade:** operações de colheita, pagamentos, vendas e estoque devem ser transacionais; correções precisam deixar rastreabilidade.
- **RNF-08 — desempenho:** telas de lançamento devem responder bem em conexão móvel comum; indicadores podem ser calculados no servidor por período.
- **RNF-09 — recuperação:** manter backups restauráveis e testar periodicamente a restauração, inclusive das chaves necessárias para ler os valores cifrados.

## Regras e definições pendentes

- Definir se a semana de pagamento começa na segunda-feira ou em outro dia.
- Confirmar se a classificação da venda é informada diretamente pelo barracão e se uma venda pode reunir vários tipos de caixa.
- Definir se o estoque admite unidades diferentes e conversões (por exemplo, litro e mililitro).
- Definir a origem dos dados das fases da Lua e se basta exibir a fase ou também o horário exato da mudança.
- Confirmar se haverá uso sem internet.
- Confirmar se os pagamentos e recebimentos parciais acontecem na prática; o desenho inicial os admite para evitar perda de informação.
- Confirmar como lançar correções feitas após uma semana já paga ou uma venda já recebida.
- Confirmar se compras de insumos podem ser feitas a prazo. O desenho inicial de saldo de caixa considera despesas registradas como já pagas.

## Fora do primeiro escopo

- Folha de pagamento formal, controle de ponto e penalização por ausência.
- Acesso de funcionários ou de operadores adicionais à mesma produção e gestão de permissões compartilhadas. O cadastro de vários clientes com contas independentes faz parte do escopo.
- Recomendação de produto, dose ou momento de aplicação de defensivos.
- Integração automática com barracão, banco, medidor de energia ou notas fiscais.
