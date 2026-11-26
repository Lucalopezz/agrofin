# Backlog Web

O backlog acompanha o que já funciona no navegador e a sequência de integração das telas. O [backlog da API](api.md) descreve contratos disponíveis e dependências. As referências de produto são os [fluxos mobile](../fluxos-mobile.md), os [requisitos](../requisitos.md) e os [critérios de aceite](../criterios-de-aceite.md).

## Entregue

- [x] **Base Web.** Landing page, navegação, painel protegido e telas organizadas por feature. React Query nos fluxos integrados, com requisições e hooks separados. A [CI](../../.github/workflows/validacao.yml) executa lint, testes e build.
- [x] **Acesso (HU-17, RF-18).** Cadastro público, login, consulta da sessão, logout e proteção do painel integrados à API. O usuário informa nome, nome de acesso e senha no cadastro.
- [x] **Funcionários (HU-01, RF-01).** Cadastro, filtro por estado, renomeação, desativação e reativação com dados persistidos. O formulário de colheita consulta funcionários ativos.
- [x] **Tarifas (HU-01 e HU-02, RF-02).** Cadastro e consulta das tarifas de caixas normais e de moagem, com vigência. A tela usa as tarifas disponíveis para apresentar a prévia de colheita.
- [x] **Lançamento e histórico de caixas (HU-02, RF-03).** Registrar caixas por funcionário e data, consultar o histórico pelo intervalo escolhido e corrigir quantidades. O histórico começa na semana atual e pode acompanhar a semana de um lançamento retroativo salvo. O cálculo persistido e as restrições de edição vêm da API.
- [x] **Base mobile (HU-15).** Ajustes de tipografia e leitura em telas estreitas, navegação compartilhada e formulários separados por responsabilidade nos fluxos já integrados. Os testes cobrem autenticação, navegação, funcionários, tarifas e colheita.
- [x] **Protótipos do restante do painel.** Há telas demonstrativas de pagamentos, vendas, despesas, insumos, pulverizações, calendário, painel e relatórios. Seus exemplos servem para revisar a interface. A persistência ou consulta real de cada fluxo está nas etapas abaixo.

## Próximas entregas

A ordem proposta segue a rotina do cliente. Fechar produção e pagamento antes de passar ao acerto com o barracão. Em seguida, integrar despesas e consolidar os resultados antes de entrar em insumos e pulverização. Usar cada etapa como unidade de entrega e deixar os encaixes de API identificados.

### 1. Fechar a semana da equipe (HU-03 a HU-05, RF-04 a RF-07)

- [ ] Integrar o lançamento de diárias à rotina de produção, com funcionário ativo, data editável e valor. Mostrar a diária separada das caixas no fluxo de conferência.
- [ ] Trocar os saldos de demonstração pelo resumo real. Começar na semana atual, permitir alterar o intervalo e enviar as mesmas datas ao consultar o resumo e ao registrar o pagamento.
- [ ] Mostrar por funcionário as caixas por tipo, o valor das caixas, as diárias, o devido, o pago e o saldo restante. Permitir voltar aos lançamentos que formam os totais, usando o histórico de caixas e avaliando se a consulta disponível atende ao detalhe de diárias.
- [ ] Registrar pagamento integral ou parcial com observação, usando a data retornada pelo servidor. Bloquear novo envio enquanto a requisição estiver em andamento e atualizar o resumo após confirmação.
- [ ] Cobrir estados vazio, carregando, erro, saldo alterado e pagamento acima do restante. Conferir o exemplo com R$ 58,00 de caixas, R$ 80,00 de diária e R$ 100,00 pagos, com saldo de R$ 38,00. Verificar no celular o caminho da colheita até a baixa do saldo.
- **API:** Registro de diária, resumo por período e registro de pagamento disponíveis. A API ainda não oferece listagem dedicada de diárias. Se os campos do resumo forem insuficientes para o detalhe exigido nos critérios de aceite, fechar essa consulta com a API nesta mesma etapa.

### 2. Registrar e conferir o acerto com o barracão (HU-06 e HU-07, RF-08 a RF-10)

- [ ] Substituir os dados locais por cadastro e consulta de vendas persistidas. Permitir informar data, caixas e preço unitário por classificação, além de desconto opcional com valor e motivo.
- [ ] Mostrar prévia do bruto, desconto e líquido antes de salvar. Usar os totais da API na confirmação e no histórico por período.
- [ ] Abrir a venda para conferir linhas, desconto, recebido e saldo a receber. Registrar recebimento parcial ou integral como operação da venda, impedir reenvio enquanto estiver pendente e atualizar a consulta após sucesso.
- [ ] Cobrir erro de desconto inválido, recebimento acima do saldo, perda de sessão e venda sem recebimento. Conferir que uma venda em um dia e um recebimento em outro continuam distinguíveis no fluxo.
- **API:** Cadastro, listagem e recebimento disponíveis. A tela de correção após recebimento depende da regra ainda em aberto.

### 3. Gerenciar despesas pagas (HU-09, RF-11)

- [ ] Integrar o formulário de despesa com categoria, data de pagamento editável, valor e descrição. Incluir energia de irrigação, fertilizantes, defensivos e outras despesas conforme o contrato.
- [ ] Consultar despesas no intervalo escolhido, abrir para editar e confirmar a exclusão. Atualizar lista e totais da tela após salvar ou excluir.
- [ ] Cobrir valor inválido, período vazio, erro de gravação e registro indisponível. Bloquear envio repetido enquanto houver uma requisição em andamento.
- **API:** Cadastro, consulta, edição e exclusão disponíveis. Compras vinculadas ao estoque entram depois, com a despesa correspondente gerada pelo fluxo de compra.

### 4. Integrar painel e relatórios por período (HU-08 a HU-10, RF-12)

- [ ] Trocar os exemplos do painel e dos relatórios por resultados reais. Usar o mesmo intervalo explícito em todos os indicadores da tela.
- [ ] Apresentar caixas colhidas por funcionário e tipo, caixas vendidas por classificação, descontos, preço médio líquido de venda, valores devidos e pagos aos funcionários e despesas pagas.
- [ ] Distinguir valor líquido vendido, valor recebido e saldo de caixa. Mostrar os detalhes disponíveis para conferir os totais e tratar períodos vazios.
- [ ] Substituir o CSV demonstrativo por dados do intervalo aplicado se a exportação existente permanecer na tela. Conferir coerência com o painel e com os lançamentos de origem.
- **API:** Depende do resumo consolidado. A parcela referente a compras de insumos entra junto com a integração financeira do estoque.

### 5. Cadastrar insumos e dar entrada em estoque (HU-09 e HU-11, RF-11, RF-13 e RF-14)

- [ ] Integrar cadastro e consulta de insumos com nome, categoria, unidade e observações.
- [ ] Registrar compra com data, quantidade, unidade e custo. Mostrar saldo atualizado e a despesa vinculada quando a compra for paga.
- [ ] Conferir que registrar uma compra atualiza o estoque uma vez e que o gasto entra uma vez no caixa. Tratar erros sem perder os dados digitados.
- **API:** Depende dos contratos de cadastro, compra e estoque. Conversão de unidades e compras a prazo precisam de confirmação.

### 6. Registrar pulverização e consultar histórico (HU-12 e HU-13, RF-15 e RF-16)

- [ ] Registrar data, observações e quantidades por insumo. Selecionar insumos existentes e mostrar a prévia dos itens antes de salvar.
- [ ] Cadastrar um insumo no próprio formulário, preservando a data e os demais campos. Permitir informar saldo inicial ou marcar saldo pendente de conferência.
- [ ] Após salvar, mostrar confirmação, evento no histórico e saldo atualizado ou pendente. Cobrir erros de gravação, unidades e saldo conforme o contrato.
- [ ] Conferir no celular o registro com insumos existentes e o caminho que exige cadastro rápido, com teclado aberto e sem rolagem horizontal.
- **API:** Depende de cadastro de insumos, registro de pulverização, histórico e baixa de estoque.

### 7. Integrar calendário de aplicação e fases da Lua (HU-14, RF-17)

- [ ] Consultar pulverizações no mês escolhido, mostrar os eventos no dia correto e abrir o detalhe ao tocar. Cobrir mês sem eventos e navegação entre meses.
- [ ] Exibir fase da Lua por dia e destacar mudanças de fase. Aplicar o fuso da operação e cobrir transições próximas ao início ou fim do mês.
- **Dependências:** Histórico de pulverizações e decisão sobre fonte lunar, fuso e precisão da transição. A API participa se a fonte exigir processamento no servidor.

### 8. Completar gestão da conta e validar a primeira semana de uso (HU-15 e HU-17)

- [ ] Integrar a alteração de nome, troca de senha e desativação da conta quando abrir a tela de configurações. Confirmar ações que encerram a sessão e retornar ao acesso quando a sessão for revogada.
- [ ] Validar no celular os atalhos para caixas, pagamentos e pulverizações. Conferir leitura com teclado aberto, prévias e confirmações, rótulos, valores padrão editáveis e repetição de um lançamento recente nos fluxos frequentes (RNF-01 a RNF-03). Conferir o carregamento e o envio dos formulários em conexão móvel (RNF-08).
- [ ] Acompanhar com o cliente uma semana de uso, da entrada de caixas à conferência dos saldos. Registrar ajustes de passos e rótulos no fluxo correspondente. Confirmar a necessidade de uso sem internet antes de definir sincronização.
- **API:** Rotas de gestão da conta disponíveis. A validação no ambiente publicado depende da publicação da API sob a mesma origem.

## Fechamento de cada fluxo

Substituir demonstração por consulta e persistência reais na etapa correspondente. Conferir o resultado após recarregar a página e com períodos vazios, lentidão, falha de envio e sessão expirada. Manter valores e saldos consistentes com a API, atualizar as consultas afetadas e testar o caminho principal em tela estreita. Usar a organização por feature e o padrão existente de requisições, hooks e componentes.
