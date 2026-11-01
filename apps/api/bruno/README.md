# Testar a API no Bruno

A coleção [agrofin.postman_collection.json](agrofin.postman_collection.json) contém as nove rotas implementadas. O formato Postman v2.1 permite importar um único arquivo no Bruno, sem instalar o Postman.

## Importar

1. No Bruno, clique em **+ → Import Collection → Postman Collection**.
2. Selecione `apps/api/bruno/agrofin.postman_collection.json`.
3. Escolha uma pasta local fora do repositório para a coleção importada, evitando versionar credenciais e cookies preenchidos no Bruno.
4. Nas variáveis da coleção, ajuste `base_url` para o endereço da API com `API_PREFIX`, `origin` para o mesmo valor de `PUBLIC_ORIGIN` no `.env`, e `login_name` e `password` para suas credenciais. Os valores do arquivo são apenas exemplos locais; a senha de cadastro exige entre 15 e 128 caracteres.

## Executar

Com a API rodando, envie as requisições **01 a 08 individualmente**, na ordem. A **02** cria uma conta de cliente/gerente sem token de instalação. Se sua conta já existir, pule essa requisição e entre com suas credenciais. Para criar outra conta, escolha um `login_name` diferente. Cada cliente acessa somente os próprios dados; funcionários não possuem login. Cada execução de **06** cria um funcionário persistido no banco.

O login entrega a sessão por cookie. Mantenha os cookies habilitados no Bruno e use o mesmo host em todas as requisições. As escritas já incluem `Origin` e `X-Agrofin-Request: 1`, exigidos pela API.

**Não execute a coleção inteira pelo runner.** A pasta **Manual - Desativar conta** contém uma operação que impede novos logins e revoga todas as sessões; a API não oferece reativação. Para testar essa operação deliberadamente, entre novamente após o logout e envie a requisição de desativação.

Para conferir o isolamento, crie um funcionário na primeira conta, saia, troque as credenciais nas variáveis e crie/entre na segunda conta. A listagem de funcionários da segunda conta começa vazia; ao voltar à primeira, seu funcionário continua disponível.
