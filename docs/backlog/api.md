# Backlog da API

O backlog acompanha o que está implementado na API Go e a ordem proposta para entregar os fluxos restantes. Os contratos em uso estão nas [rotas Go](../../apps/api/internal) e na [coleção HTTP](../../apps/api/bruno/agrofin.postman_collection.json). O escopo vem dos [requisitos](../requisitos.md), das [histórias de usuário](../historias-de-usuario.md) e dos [critérios de aceite](../criterios-de-aceite.md). A integração das telas está no [backlog Web](web.md).

## Entregue

- [x] **Base local.** API Go, PostgreSQL, migrações versionadas, health check, prefixo configurável e proxy Web para a mesma origem no Compose. A [CI](../../.github/workflows/validacao.yml) executa análise Go, migrações, testes com PostgreSQL e verificação do Compose.
- [x] **Conta e sessão (HU-17, RF-18).** Cadastro público com nome de acesso único, senha com Argon2id, login, logout, consulta da sessão, alteração de nome e senha e desativação da conta. Sessão em cookie e validação de origem nas escritas. O acesso aos registros é restrito ao cliente autenticado. A reativação de conta desativada depende de operação manual.
- [x] **Funcionários (HU-01, RF-01).** Cadastrar pelo nome, listar todos ou filtrar por ativos/inativos, renomear, desativar e reativar o mesmo registro. Preservar vínculos e histórico quando o estado mudar.
- [x] **Tarifas (HU-01 e HU-02, RF-02).** Criar e listar tarifas gerais por conta, tipo de caixa e vigência, com valor cifrado. O início é inclusivo e o fim, quando informado, é exclusivo. O banco protege contra vigências incompatíveis para o mesmo tipo.
- [x] **Caixas colhidas (HU-02, RF-03 e RF-07).** Registrar funcionário, data, tipo e quantidade, preservar a tarifa aplicada, listar por período e consultar o resumo de caixas por funcionário. Corrigir itens preservando funcionário, data e tarifas fotografadas, com bloqueio se afetar período quitado ou reduzir o devido abaixo do valor já pago. Dias sem lançamentos não geram faltas.
- [x] **Diárias e pagamento (HU-03 a HU-05, RF-04 a RF-06).** Registrar diárias separadas de caixas, consultar devido, pago e restante por funcionário e registrar pagamento integral ou parcial com data do servidor e observação. O resumo e a baixa usam o mesmo intervalo inclusivo, informado pelo cliente HTTP. O saldo impede pagamento acima do devido. Cada envio aceito cria um pagamento.
- [x] **Venda e recebimento (HU-06 e HU-07, RF-08 a RF-10).** Registrar e listar vendas com linhas de exportação, mercado e moagem, preço por caixa e desconto opcional com motivo. Calcular bruto, líquido, recebido e saldo a receber. Registrar recebimentos até o saldo da venda. A listagem apresenta os recebimentos agregados e cada envio aceito cria um recebimento.
- [x] **Despesas gerais pagas (HU-09, RF-11).** Cadastrar, listar por período, editar e excluir despesas com data de pagamento, categoria, valor e descrição opcional. O valor é cifrado. A edição e a exclusão já afetam os resultados da listagem. O histórico de alterações está pendente.
- [x] **Base de proteção financeira (HU-16, RNF-04 a RNF-07).** A aplicação cifra tarifas, valores aplicados às colheitas, diárias, pagamentos, preços de venda, descontos, recebimentos e despesas antes da gravação. Os dados são filtrados por conta antes da leitura dos valores. O chaveiro e a versão ativa são fornecidos ao processo fora do banco.
- [x] **Esquema inicial de insumos e auditoria.** Há migrações para insumos, compras, pagamentos de compras, pulverizações, movimentos de estoque e eventos de auditoria. A presença das tabelas cobre a preparação do esquema. Os fluxos de aplicação estão abaixo. [Migrações](../../apps/api/migrations)
- [x] **Resumo de produção e caixa (HU-08 a HU-10, RF-12).** Consolida colheitas, vendas, descontos, preço médio líquido, devido à equipe, pagamentos, recebimentos e despesas pagas por período. As consultas de conferência preservam as datas de origem. [Contrato e conferência](../../apps/api/bruno/README.md#produção-e-caixa-por-período)

## Próximas entregas

O painel já consolida os lançamentos disponíveis. A compra paga entra nos indicadores quando sua origem financeira for implementada.

### 1. Cadastrar insumos e registrar compras com entrada em estoque (HU-09 e HU-11, RF-11, RF-13 e RF-14)

- [ ] Entregar cadastro, consulta e identificação de insumos por conta, com nome, categoria, unidade e observações.
- [ ] Registrar compra com data, insumo, quantidade, unidade e custo cifrado. Gravar a entrada no estoque junto com a compra e expor o saldo para consulta.
- [ ] Vincular a compra paga ao caixa uma única vez. Usar uma única origem financeira para o gasto e garantir que consultar ou consumir o estoque não gere outra despesa. Cobrir a inclusão da compra no resumo por período.
- [ ] Proteger a compra e seus efeitos contra falha parcial e repetição da gravação. Validar com o cliente se existem compras a prazo e se há necessidade de conversões de unidade antes de definir esses contratos. [Modelo de dados](../modelo-de-dados.md)

### 2. Registrar pulverização, baixar estoque e consultar histórico (HU-12 e HU-13, RF-15 e RF-16)

- [ ] Registrar data, observações, insumos e quantidades usadas. Consultar histórico por período e recuperar os detalhes de cada evento da conta.
- [ ] Persistir evento, itens e baixas de estoque em conjunto. Testar integridade em caso de erro, repetição e operações concorrentes.
- [ ] Suportar cadastro rápido no fluxo. Permitir saldo inicial informado ou estoque pendente de conferência quando o saldo anterior for desconhecido. Registrar o uso no histórico sem criar uma compra para justificá-lo.
- [ ] Validar a relação entre as unidades de compra, saldo e uso conforme a decisão do cadastro de compras. Confirmar o comportamento de estoque conhecido insuficiente antes de fechar a regra.

### 3. Disponibilizar os dados do calendário mensal (HU-14, RF-17)

- [ ] Usar as pulverizações persistidas no histórico como fonte para o mês, com acesso ao detalhe de cada evento.
- [ ] Definir com a Web onde obter os dados lunares, como aplicar o fuso da operação e se o horário de transição precisa ser exibido. Entregar o suporte na API se a fonte escolhida exigir processamento no servidor.
- [ ] Conferir fase por dia e datas de transição, inclusive nos limites do mês e em horários que mudam de dia entre fusos.

### 4. Fechar regras pendentes de venda e histórico financeiro

- [ ] Validar com o cliente a classificação enviada pelo barracão, vendas com múltiplos tipos e a ocorrência de pagamentos e recebimentos parciais. Os contratos atuais já aceitam esses casos.
- [ ] Definir como corrigir venda após recebimento. Só então implementar o fluxo, com testes para líquido, saldo, recebimentos já registrados e impacto no caixa.
- [ ] Definir o histórico de alterações financeiras a partir das exigências operacionais. A migração de auditoria existe e ainda precisa de integração aos fluxos escolhidos. Conferir se dados sensíveis do histórico usam o mesmo padrão de proteção.
- [ ] Confirmar se o uso sem internet é necessário. Se for, definir com a Web as regras de sincronização, repetição de envios e conflitos antes de criar contratos novos.

### 5. Validar publicação e recuperação (HU-16, RNF-05, RNF-06 e RNF-09)

- [ ] Validar um protótipo React + Go sob a mesma origem na Vercel, conforme o [plano](../plano-de-implementacao.md). O protótipo pode ser antecipado para esclarecer a viabilidade do deploy. Conferir rotas, proxy, sessão, HTTPS, migrações e acesso ao PostgreSQL antes de definir a configuração final.
- [ ] Definir o provedor de produção e como fornecer, versionar, rotacionar e recuperar chaves financeiras separadas do banco. Definir o controle operacional do volume de cifras por chave.
- [ ] Testar restauração do banco e das chaves em ambiente separado. Conferir leitura dos registros cifrados restaurados, isolamento entre contas e ausência de valores sensíveis nos logs e em texto claro na cópia do banco.
- [ ] Validar as jornadas principais no ambiente publicado e acompanhar a primeira semana de uso junto com a Web. [Sequência de implementação](../plano-de-implementacao.md#sequência-sugerida)

## Fechamento de cada fluxo

Entregar contrato HTTP, regras, persistência e migração quando necessária no mesmo fluxo. Cobrir os casos de aceite correspondentes, incluindo isolamento por conta, integridade sob concorrência e proteção dos valores. Atualizar a coleção Postman v2.1 com as novas requisições e manter operações com efeitos relevantes de conta ou sessão na pasta manual. Registrar mudanças de contrato antes de iniciar o próximo fluxo.
