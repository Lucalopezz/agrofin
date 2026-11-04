# Critérios de aceite

Estes exemplos tornam as histórias verificáveis. Valores são ilustrativos; não definem tarifas reais.

## Contas e acesso

### HU-17 — cadastro público e isolamento

- Dois clientes podem criar contas com nomes de acesso diferentes sem sessão prévia, convite ou token de instalação, mesmo que tenham o mesmo nome de exibição.
- Repetir um nome de acesso, inclusive variando maiúsculas e espaços externos, retorna `409` sem alterar a conta existente.
- Cada cliente entra com suas próprias credenciais e recebe uma sessão vinculada ao seu usuário.
- Funcionários cadastrados por um gerente não aparecem para outro; enviar `user_id` no corpo ou na query não permite escolher o dono dos registros.
- Desativar uma conta revoga somente suas sessões; outras contas continuam funcionando.

## Colheita, diárias e pagamentos

### HU-01 e HU-02 — funcionários e caixas

- O cadastro pede apenas o nome. O sistema gera o identificador usado para relacionar colheitas e pagamentos, mesmo quando houver nomes iguais.
- Cada cliente entra com a própria conta; cadastrar um funcionário não cria usuário, senha ou convite.
- Renomear um funcionário remove espaços das extremidades, rejeita nome vazio e preserva ID, dono, estado ativo/inativo e data de desativação. Nomes iguais são permitidos.
- Uma conta não pode renomear funcionários de outra; a resposta é a mesma de um funcionário inexistente e nenhum registro é alterado.
- Dado um funcionário ativo e uma tarifa vigente de R$ 5,00 por caixa normal, ao registrar 10 caixas normais em um dia, o lançamento mostra R$ 50,00.
- Se a tarifa mudar para R$ 6,00 no dia seguinte, o lançamento anterior continua valendo R$ 50,00.
- Uma colheita exige data, funcionário, tipo e quantidade maior que zero. A edição registra quem alterou, quando alterou e os valores anteriores.
- Desativar um funcionário marca o cadastro como inativo e registra a data, preservando ID, dono, nome e histórico. A listagem do cadastro continua retornando o registro com `active=false`.
- Repetir a desativação retorna o mesmo sucesso sem alterar a primeira data. Funcionário inexistente ou de outra conta recebe a mesma resposta `404`, sem alterar registros.
- Funcionários desativados deixam de aparecer na lista principal de novo lançamento, mas continuam visíveis no histórico.
- Na semana 1, lançamentos de X e Z fazem ambos aparecer no resumo. Na semana 2, lançamentos de Z e F fazem apenas Z e F aparecer no resumo daquela semana. X continua no histórico e não é desativado automaticamente.

### HU-02 — criação inicial de lançamento implementada

`POST /api/v1/harvest-entries` cria o cabeçalho e seus itens em uma transação. A consulta inicial de lançamentos está disponível em `GET /api/v1/harvest-entries`; edição e histórico de correções continuam pendentes.

- A sessão define o dono do lançamento. O corpo contém somente `employee_id`, `harvest_date` e `items`; campos extras, UUID inválido, data fora de `AAAA-MM-DD` e quantidade fora do intervalo de inteiro positivo são rejeitados.
- O funcionário precisa pertencer à conta e estar ativo. Registro ausente ou de outra conta retorna `404`; funcionário inativo retorna `409`.
- O lançamento aceita um item normal, um item de moagem ou ambos. Tipos desconhecidos, quantidade zero ou negativa e tipos repetidos retornam `400`.
- Para cada tipo, usa a tarifa geral vigente na data da colheita. Sem tarifa vigente para um dos tipos, retorna `409` e não grava parcialmente o lançamento.
- Cabeçalho e itens são persistidos juntos. Se a gravação de qualquer item falhar, nenhum cabeçalho parcial permanece.
- Cada tarifa aplicada é gravada cifrada como fotografia do item; mudar tarifas futuras não recalcula esse lançamento. A resposta `201` devolve os tipos, quantidades e tarifas unitárias aplicadas à sessão autenticada, com `Cache-Control: no-store`.
- A verificação de estado do funcionário permanece bloqueada até a conclusão da gravação, evitando aceitar uma colheita que concorra com a desativação.
- Não há chave de idempotência nesta etapa; repetir uma requisição aceita pode criar outro lançamento. Correção com auditoria e tratamento de repetição continuam pendentes.

### HU-02 — consulta inicial de lançamentos implementada

`GET /api/v1/harvest-entries` exige sessão e recebe `start_date` e `end_date` em `AAAA-MM-DD`. Os dois limites são inclusivos; informar ambos evita que uma tela consulte todo o histórico sem delimitação. Esta é a forma inicial do filtro da API, sem definir o começo da semana de pagamento.

- A sessão define o gerente dono da consulta. A API não aceita `user_id` ou `employee_id` como substitutos de autorização; os resultados incluem apenas lançamentos pertencentes ao gerente autenticado.
- Os valores persistidos da tarifa aplicada são lidos cifrados. Depois do filtro por dono e período, o serviço autentica e descriptografa cada fotografia para a sessão dona, retornando `applied_rate_cents` por item.
- O resultado mantém cada lançamento e seus itens, ordenados por data e criação mais recentes; não calcula totais semanais nem junta o cadastro completo de funcionários ao agregado.
- Uma faixa vazia retorna `[]`. Parâmetros ausentes, repetidos, extras ou malformados e fim anterior ao início retornam `400`; sem sessão retorna `401`.
- Se a consulta ao banco ou a autenticação de qualquer envelope falhar, a API retorna erro sem entregar uma lista parcial. Respostas contêm dados financeiros e usam `Cache-Control: no-store`.

### HU-01 e HU-02 — núcleo local de tarifas

Estes critérios verificam o domínio em memória, já implementado. O serviço, a proteção local e o repositório PostgreSQL também estão testados; configuração de chaves no runtime e a rota HTTP de criação estão implementadas.

- Normal e moagem são tipos válidos. Tipo vazio, desconhecido ou classificação de venda é rejeitado.
- O valor é um inteiro de centavos maior que zero; `1` centavo é válido e valores zero ou negativos são rejeitados.
- A data inicial é obrigatória. Fim ausente representa vigência aberta; fim informado como data zero é inválido.
- As datas são tratadas como dias de calendário, preservando o dia no fuso recebido. Horários são removidos antes da comparação; o fim deve estar em um dia posterior ao início, com limite final exclusivo.
- Normalizar não altera a entrada nem compartilha o ponteiro da data final. Uma mudança posterior nesse ponteiro não modifica o resultado validado.
- A validação local preserva os identificadores técnicos. O cadastro usa tarifa geral por conta e tipo, igual para todos os funcionários na mesma vigência; não recebe funcionário ou dono no input.

### HU-01 e HU-02 — serviço e proteção local de tarifas

Estes critérios são verificados com dependências controladas e cifra real. A integração PostgreSQL está detalhada na seção seguinte; o cadastro HTTP usa a sessão para obter o dono e não aceita campos de identidade do cliente.

- O serviço valida antes de consultar ou cifrar, usa o gerente do chamador confiável e rejeita sobreposição antes da escrita. A persistência concreta reforça essa regra sob concorrência.
- O ID vem da interface de persistência antes da cifra; o serviço entrega uma `ProtectedRate` sem centavos em claro. Falhas e cancelamento não devolvem uma entidade parcialmente cadastrada nem avançam para escrita indevida.
- A cifra autenticada recupera valores positivos em todo o intervalo de `int64`, incluindo um centavo. Valores iguais recebem nonces novos.
- Conta ou registro diferentes, envelope alterado, versão desconhecida e material de chave incorreto falham sem devolver centavos ou incluir dados privados no erro.
- Versões antigas podem ser recuperadas quando sua chave é fornecida; novas cifras usam a versão ativa. A ausência da chave antiga falha sem tentar outra versão silenciosamente.
- Chaves e centavos não fazem parte do envelope persistível. Configuração inválida ou repetição de material sob versões diferentes é rejeitada.
- Provisionamento e controle global dos limites por chave ainda precisam ser integrados antes de habilitar escrita em produção.

### HU-01 e HU-02 — persistência PostgreSQL de tarifas

Estes critérios estão cobertos por integração com banco migrado, serviço e cifra reais. A composição no runtime e as rotas HTTP `GET` e `POST /harvest-rates` estão implementadas.

- O ID é gerado no PostgreSQL antes da cifra, sem criar linha parcial. A gravação usa o mesmo ID e persiste somente envelope e metadados, com `employee_id` nulo.
- O envelope lido do banco é recuperável com a chave histórica correspondente e o mesmo gerente/registro; chave incorreta ou gerente trocado falham.
- A sobreposição considera somente conta, tipo e escopo geral. Períodos adjacentes são permitidos e uma vigência sem fim bloqueia intervalos posteriores sobrepostos do mesmo escopo.
- Duas criações que passaram pela consulta de sobreposição disputam a restrição real: somente uma persiste e a outra recebe `ErrRateOverlap`.
- Outros erros de integridade, como ID duplicado, não são traduzidos como sobreposição. Cancelamento e prazo esgotado continuam identificáveis nos métodos do repositório.

### HU-01 e HU-02 — criação HTTP de tarifas

- `POST /api/v1/harvest-rates` exige sessão válida, origem permitida e `X-Agrofin-Request: 1`; sem sessão, responde `401`.
- O corpo contém somente `box_type`, `amount_cents`, `valid_from` e `valid_until`; rejeita campos extras como `user_id` e `employee_id`. Datas usam `AAAA-MM-DD`, e `valid_until: null` representa vigência aberta.
- O serviço usa o gerente da sessão. Não é possível criar uma tarifa em outra conta escolhendo um identificador no corpo.
- Uma tarifa válida retorna `201` com ID e os dados da tarifa para a sessão autenticada. `amount_cents` é persistido somente cifrado.
- Tipo, centavos, vigência, JSON ou data inválidos retornam `400`; sobreposição do mesmo tipo e conta retorna `409`.
- Se a configuração das chaves AES-256 estiver ausente ou inválida, a API encerra a inicialização sem disponibilizar a rota de escrita.
- A pasta Manual correspondente na coleção de formato Postman v2.1 permite conferir a criação após entrar. O arquivo pode ser usado no Postman ou importado no Bruno. A listagem retorna todas as tarifas da própria conta, com valores descriptografados somente para a sessão autenticada.

### HU-01 e HU-02 — listagem HTTP de tarifas

- `GET /api/v1/harvest-rates` exige sessão válida e retorna apenas tarifas gerais da conta autenticada, sem aceitar um `user_id` fornecido pelo cliente.
- A consulta inclui o histórico e ordena por início de vigência mais recente, tipo e ID. Uma conta sem tarifas recebe `200` com `[]`.
- O serviço descriptografa cada valor usando o gerente autenticado e o ID do registro. Se um envelope não puder ser autenticado, não devolve uma lista parcial nem registra valores financeiros; responde `500` com mensagem genérica.
- A resposta contém ID, tipo, centavos e datas. `Cache-Control: no-store` evita armazenar valores financeiros no navegador ou em proxies.
- **07.3 - Listar tarifas** na coleção Postman v2.1 permite conferir o fluxo com sessão ativa; a requisição Manual de criação permite criar uma tarifa para visualizar na listagem. A coleção pode ser aberta no Postman ou importada no Bruno.

### HU-01 — reativação e filtro implementados

A reativação e o filtro de estado estão implementados na API; sua integração com a interface continua pendente.

- Reativar restaura o mesmo cadastro com `active=true` e `deactivated_at` vazio, preservando ID, dono, nome e vínculos históricos. Não cria outro funcionário nem altera lançamentos anteriores.
- Repetir a reativação retorna `204` sem corpo, inclusive para funcionário já ativo. Registro inexistente ou de outra conta recebe `404`, sem alterar registros nem revelar sua existência.
- Desativar novamente depois de reativar registra uma nova data para o novo ciclo de inatividade. A repetição de uma desativação no mesmo ciclo mantém a primeira data.
- O filtro `active=true` retorna somente ativos; `active=false`, somente inativos. Sem filtro, a listagem continua retornando ambos, sempre limitados ao gerente da sessão, ordenados por nome e ID e sem deduplicar nomes. Sem resultados, retorna `[]`.
- `active` vazio, inválido, repetido, mesmo com valores iguais, ou query malformada retorna `400` antes de consultar a persistência. O filtro não aceita `1`, `0` ou variações em maiúsculas.
- Após desativar ou reativar um funcionário, a próxima consulta filtrada reflete o estado persistido sem criar outro registro ou misturar dados de contas diferentes.

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
