# Usar a coleção HTTP da API

O arquivo [agrofin.postman_collection.json](agrofin.postman_collection.json) contém as requisições da API e segue o formato Postman Collection v2.1. Ele pode ser importado diretamente no Postman, no Bruno ou no Insomnia: o [Postman importa coleções exportadas](https://learning.postman.com/docs/getting-started/importing-and-exporting/importing-data/), o [Bruno importa coleções Postman](https://docs.usebruno.com/get-started/import-export-data/import-collections) e o [Insomnia declara suporte a Postman v2.0/v2.1](https://developer.konghq.com/insomnia/import-export/).

O arquivo descreve requisições HTTP; não é um script JavaScript específico de um cliente.

O repositório mantém um único arquivo compartilhado no caminho `apps/api/bruno`, embora o formato seja Postman Collection e possa ser usado em mais de um cliente. Os passos abaixo cobrem Postman e Bruno. No Insomnia, a importação é compatível, mas confira as variáveis e o armazenamento de cookies depois que o cliente converter a coleção.

## Importar

1. No Postman, clique em **Import** e selecione `apps/api/bruno/agrofin.postman_collection.json`. No Bruno, use **+ → Import Collection → Postman Collection** e selecione o mesmo arquivo.
2. No Insomnia, use **Import → File**, selecione o arquivo e conclua a importação após revisar o que será criado.
3. Nas variáveis da coleção ou no ambiente equivalente, ajuste `base_url` para o endereço da API com `API_PREFIX`, `origin` para o mesmo valor de `PUBLIC_ORIGIN` no `.env`, e `login_name` e `password` para suas credenciais. Para testar a troca de senha, defina também `new_password` para um valor diferente e válido. Os valores versionados são exemplos locais; não salve credenciais reais no repositório. A senha exige entre 15 e 128 caracteres.

## Executar

Com a API rodando, envie as requisições **01 a 11 individualmente**, na ordem. A **02** cria uma conta de cliente/gerente sem token de instalação. Se sua conta já existir, pule essa requisição e entre com suas credenciais. Para criar outra conta, escolha um `login_name` diferente. Cada cliente acessa somente os próprios dados; funcionários não possuem login. Cada execução de **06** cria um funcionário persistido no banco. Antes de executar **08**, **09**, **10** ou registrar uma colheita, copie o `id` retornado por **06** ou **07** para a variável `employee_id`. A renomeação exige sessão, aceita somente `name` e preserva o estado ativo/inativo. Execute **07** novamente para conferir. A requisição **09** desativa o funcionário de `employee_id`, sem corpo, e retorna `204` vazio. Na listagem **07**, o registro continua disponível com `active=false`. Repetir **09** também retorna `204` e preserva a primeira data de desativação. A requisição **10** reativa o mesmo cadastro, também sem corpo e com resposta `204`: execute **07** novamente para confirmar `active=true`. Repetir a reativação também é sucesso. Envie **07.3** com a sessão ativa para consultar as tarifas, **07.4** para consultar lançamentos e **07.5** para resumir a produção por funcionário, ajustando as datas na faixa de variáveis da coleção. Execute **11** depois de concluir as consultas e as criações manuais.

Para criar uma tarifa, entre e envie manualmente **Manual - Criar tarifa → Cadastrar tarifa geral** antes de sair. O corpo aceita `box_type` (`normal` ou `moagem`), `amount_cents` inteiro positivo, `valid_from` em `AAAA-MM-DD` e `valid_until` na mesma forma ou `null`. O gerente vem da sessão. A resposta `201` retorna ID, tipo, valor e vigência; tipo ou valor inválido, ou data malformada, retorna `400`; tarifa sobreposta do mesmo tipo na conta retorna `409`. A criação grava uma linha persistente, então ajuste o tipo ou escolha uma vigência adjacente ao repetir. A API exige as variáveis de chave descritas no README principal antes de iniciar.

Para consultar as tarifas, envie **07.3 - Listar tarifas** com uma sessão ativa. A resposta contém o histórico da conta, com os valores descriptografados para o gerente autenticado e sem incluir tarifas individuais. A ordem começa pela data de início mais recente; sem registros, retorna `[]`. Uma falha de leitura ou de autenticação de qualquer envelope retorna `500` sem dados parciais.

Para consultar lançamentos, envie **07.4 - Listar lançamentos de colheita** com uma sessão ativa. Ajuste `harvest_start_date` e `harvest_end_date` nas variáveis da coleção; ambas usam `AAAA-MM-DD`, e o intervalo inclui os dois dias. A resposta lista os lançamentos do gerente autenticado, com funcionário, data, tipo, quantidade e tarifa unitária aplicada em centavos. Sem resultados, retorna `[]`. Datas ausentes, repetidas, inválidas, parâmetros extras ou `end_date` anterior a `start_date` retornam `400`. Uma falha de leitura ou de autenticação de qualquer tarifa aplicada retorna `500` sem lista parcial.

Para consultar o resumo por funcionário, envie **07.5 - Resumir colheitas por funcionário** com a mesma faixa de datas de **07.4**. A resposta agrupa quantidade e valor gerado pelas caixas normais e de moagem. O valor total considera a tarifa fotografada em cada lançamento; ele não desconta pagamentos nem inclui diárias ou ajustes, então ainda não é o saldo semanal de pagamento. Sem lançamentos, retorna `[]`.

Para registrar uma colheita, mantenha a sessão ativa, copie o ID de um funcionário ativo para `employee_id` e envie **Manual - Registrar colheita → Registrar caixas do funcionário**. Antes, cadastre uma tarifa geral vigente na data para cada tipo que incluir no corpo; se faltar uma, a resposta é `409` e nada é gravado. O exemplo inclui normal e moagem, cada qual com quantidade inteira positiva. A resposta `201` mostra o ID do lançamento e a tarifa unitária aplicada em centavos. O funcionário precisa pertencer à conta autenticada. A criação é persistente e não tem chave de idempotência; evite repetir o envio após uma resposta de sucesso. Depois, confira o registro pela requisição **07.4**, ajustando a faixa de datas para incluir `harvest_date`.

O login entrega a sessão por cookie. Mantenha o armazenamento de cookies habilitado no cliente HTTP e use o mesmo host em todas as requisições. As escritas já incluem `Origin` e `X-Agrofin-Request: 1`, exigidos pela API.

**Não execute a coleção inteira pelo runner.** As pastas **Manual - Trocar senha**, **Manual - Desativar conta**, **Manual - Criar tarifa** e **Manual - Registrar colheita** contêm operações persistentes. A troca de senha revoga todas as sessões; depois dela, atualize a variável `password` com o valor de `new_password` e faça login novamente. A desativação impede novos logins e a reativação exige intervenção de operador. Para testar qualquer uma delas, entre novamente após o logout e envie somente a requisição desejada.

Para conferir o isolamento, crie um funcionário na primeira conta, saia, troque as credenciais nas variáveis e crie/entre na segunda conta. A listagem de funcionários da segunda conta começa vazia; ao voltar à primeira, seu funcionário continua disponível.

Para conferir a proteção da renomeação, tente usar na segunda conta o `employee_id` criado pela primeira: a resposta deve ser `404`, igual à de um ID inexistente. Um UUID malformado retorna `400`. O funcionário da primeira conta deve permanecer com o nome anterior.

A desativação também respeita o isolamento: usar o `employee_id` de outra conta retorna `404` e não altera aquele registro, mesmo que ele já esteja inativo. UUID malformado ou qualquer corpo na requisição retorna `400`. O `DELETE` exige `Origin` e `X-Agrofin-Request: 1`, mas não exige `Content-Type`.

Para testar a reativação isoladamente, faça login, preencha `employee_id` e envie **10 - Reativar funcionário**. Ela exige `Origin` e `X-Agrofin-Request: 1`, sem corpo ou necessidade de `Content-Type`. UUID ou corpo inválido retorna `400`; registro inexistente ou de outra conta retorna `404`. Reativar limpa a data da inatividade atual; desativar novamente inicia um novo ciclo com outra data. Isso não reativa contas de gerente nem altera sessões. O filtro de ativos/inativos está disponível nas requisições **07.1** e **07.2**.

## Filtrar funcionários por estado

- **07 - Listar funcionários:** sem `active`, retorna ativos e inativos.
- **07.1 - Listar funcionários ativos:** usa `?active=true`.
- **07.2 - Listar funcionários inativos:** usa `?active=false`.

As três consultas exigem sessão e retornam somente dados do gerente autenticado, ordenados por nome e ID; sem resultados, retornam `[]`. Envie `active` uma única vez, com `true` ou `false` em minúsculas. Valor vazio, diferente ou repetido e query malformada retornam `400`.

Para conferir, cadastre um funcionário e execute **07.1**; depois desative-o com **09** e execute **07.2**. Reative-o com **10** e confirme que ele volta a aparecer em **07.1**. A consulta **07** continua mostrando o mesmo cadastro nos dois estados. Tente o filtro nas duas contas para conferir o isolamento; enviar `user_id` na query não altera o dono derivado da sessão.
