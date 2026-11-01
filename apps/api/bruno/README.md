# Testar a API no Bruno

A coleção [agrofin.postman_collection.json](agrofin.postman_collection.json) contém as doze rotas implementadas. O formato Postman v2.1 permite importar um único arquivo no Bruno, sem instalar o Postman.

## Importar

1. No Bruno, clique em **+ → Import Collection → Postman Collection**.
2. Selecione `apps/api/bruno/agrofin.postman_collection.json`.
3. Escolha uma pasta local fora do repositório para a coleção importada, evitando versionar credenciais e cookies preenchidos no Bruno.
4. Nas variáveis da coleção, ajuste `base_url` para o endereço da API com `API_PREFIX`, `origin` para o mesmo valor de `PUBLIC_ORIGIN` no `.env`, e `login_name` e `password` para suas credenciais. Para testar a troca de senha, defina também `new_password` para um valor diferente e válido. Os valores do arquivo são apenas exemplos locais; a senha exige entre 15 e 128 caracteres.

## Executar

Com a API rodando, envie as requisições **01 a 10 individualmente**, na ordem. A **02** cria uma conta de cliente/gerente sem token de instalação. Se sua conta já existir, pule essa requisição e entre com suas credenciais. Para criar outra conta, escolha um `login_name` diferente. Cada cliente acessa somente os próprios dados; funcionários não possuem login. Cada execução de **06** cria um funcionário persistido no banco. Antes de executar **08** ou **09**, copie o `id` retornado por **06** ou **07** para a variável `employee_id`. A renomeação exige sessão, aceita somente `name` e preserva o estado ativo/inativo. Execute **07** novamente para conferir. A requisição **09** desativa o funcionário de `employee_id`, sem corpo, e retorna `204` vazio. Na listagem **07**, o registro continua disponível com `active=false`. Repetir **09** também retorna `204` e preserva a primeira data de desativação. Execute **10** para sair.

O login entrega a sessão por cookie. Mantenha os cookies habilitados no Bruno e use o mesmo host em todas as requisições. As escritas já incluem `Origin` e `X-Agrofin-Request: 1`, exigidos pela API.

**Não execute a coleção inteira pelo runner.** As pastas **Manual - Trocar senha** e **Manual - Desativar conta** contêm operações que alteram o acesso à conta. A troca de senha revoga todas as sessões; depois dela, atualize a variável `password` com o valor de `new_password` e faça login novamente. A desativação impede novos logins e a reativação exige intervenção de operador. Para testar qualquer uma delas, entre novamente após o logout e envie somente a requisição desejada.

Para conferir o isolamento, crie um funcionário na primeira conta, saia, troque as credenciais nas variáveis e crie/entre na segunda conta. A listagem de funcionários da segunda conta começa vazia; ao voltar à primeira, seu funcionário continua disponível.

Para conferir a proteção da renomeação, tente usar na segunda conta o `employee_id` criado pela primeira: a resposta deve ser `404`, igual à de um ID inexistente. Um UUID malformado retorna `400`. O funcionário da primeira conta deve permanecer com o nome anterior.

A desativação também respeita o isolamento: usar o `employee_id` de outra conta retorna `404` e não altera aquele registro, mesmo que ele já esteja inativo. UUID malformado ou qualquer corpo na requisição retorna `400`. O `DELETE` exige `Origin` e `X-Agrofin-Request: 1`, mas não exige `Content-Type`.
