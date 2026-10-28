# plano de implementação

## arquitetura proposta

Um monorepo com `apps/web` (React + TypeScript) e `apps/api` (Go). A interface consome `/api` no mesmo domínio. Banco relacional para lançamentos, estoque e histórico; escolher o provedor ao iniciar a implementação. O deploy desejado é um único projeto Vercel com frontend e API publicados juntos por Vercel Services, que está em beta. Validar um protótipo mínimo de build e roteamento do serviço Go antes de implementar as regras de negócio. [Fonte: Vercel Services](https://vercel.com/docs/services) e [runtime Go](https://vercel.com/docs/functions/runtimes/go).

```text
agrofin/
  apps/
    api/
      go.mod
      cmd/api/                 # entrada local
      cmd/migrate/             # aplicar e consultar migrações
      internal/
        colheita/
        funcionarios/          # cadastro simples e rotas da feature
        pagamentos/
        vendas/
        despesas/
        insumos/
        pulverizacoes/
        relatorios/
        platform/             # banco, autenticação, criptografia
      migrations/
    web/
      src/
        app/                  # rotas e composição
        features/
          user/
          limao/
          funcionarios/
          pagamentos/
          vendas/
          despesas/
          insumos/
          pulverizacoes/
          calendario/
          relatorios/
        shared/               # componentes e utilidades comuns
  docs/
  vercel.json                 # configuração após validar o protótipo
```

No Go, manter `cmd` para a entrada executável e `internal` para os pacotes do projeto. As rotas de uma funcionalidade pertencem ao pacote dessa funcionalidade, como `internal/funcionarios`; `cmd/api` compõe as rotas e inicia o servidor. `platform` recebe infraestrutura transversal apenas quando ela surgir. Os pacotes devem refletir responsabilidades reais, sem impor camadas vazias. Uma adaptação de entrada para a Vercel pode chamar a mesma aplicação HTTP usada localmente. No React, organizar por `features/<domínio>`, seguindo o padrão informado pelo usuário; código compartilhado fica em `shared` quando for realmente reutilizado.

O PostgreSQL local, as migrações e a conexão inicial da API já estão implementados. A autenticação, a escrita dos fluxos de negócio via HTTP, a criptografia na aplicação e o protótipo de deploy unificado ainda pertencem às próximas etapas.

### organização interna do cadastro de funcionários

O caminho interno de criação está implementado em `internal/funcionarios`, com todos os arquivos no mesmo pacote:

| arquivo | responsabilidade implementada |
| --- | --- |
| `employee.go` | Define `Employee` e normaliza o nome, removendo espaços das extremidades e rejeitando resultado vazio. Preserva acentos, grafia e espaços internos. |
| `service.go` | `Service.Create` valida o nome e monta a entidade `Employee` com conta e estado ativo inicial antes de chamar a persistência. |
| `repository.go` | `Repository.Create` recebe um `Employee` validado, executa o `INSERT` parametrizado e devolve a entidade com o ID gerado pelo banco. |
| `routes.go` | Mantém o GET provisório que retorna uma lista vazia. Ainda não chama o serviço. |

A composição do fluxo interno é:

```go
repo := funcionarios.NewRepository(pool)
service := funcionarios.NewService(repo)
employee, err := service.Create(ctx, accountID, name)
```

O pool é aberto e fechado pela aplicação; o repositório apenas o utiliza. A interface privada `employeeCreator`, declarada junto do serviço que a consome, permite testar a validação sem PostgreSQL. Ela descreve a única operação necessária, sem criar um repositório genérico ou interfaces antecipadas para outros domínios. Cada funcionalidade pode ter seu próprio repositório usando o mesmo pool.

O serviço recebe conta e nome, normaliza e valida o nome e monta `Employee{AccountID: accountID, Name: name, Active: true}`. Em seguida chama `repo.Create(ctx, employee)`. O repositório recebe a struct de domínio por valor, persiste conta, nome e estado e devolve a entidade com ID gerado pelo banco. Receber por valor evita alterar a struct do chamador implicitamente; o resultado persistido é explícito no retorno. O ID fica vazio somente antes da inserção. `AccountID` é um vínculo técnico do domínio, não um campo do formulário HTTP. Nomes iguais continuam permitidos. O repositório não repete a normalização nem escolhe o estado inicial; as restrições SQL continuam protegendo a integridade. Como os campos de `Employee` são exportados, a struct sozinha não comprova validação: o serviço é a entrada das operações de cadastro. A inserção atual é uma única instrução atômica, sem necessidade de transação explícita.

`ctx context.Context` é passado como primeiro argumento do serviço até `QueryRow`, propagando cancelamento e prazo. Não é guardado nas structs. No futuro handler, ele virá de `r.Context()`, e não de um novo `context.Background()`. O nome e o `accountID` continuam sendo argumentos explícitos; contexto não substitui os dados do cadastro.

**Ainda pendente:** handler de criação, rota POST, composição no `main`, origem autenticada de `accountID`, listagem persistida, renomeação e desativação. O handler lerá o JSON e traduzirá resultados para HTTP; as regras continuarão no serviço e o SQL no repositório. `accountID` deve identificar uma conta existente e vir de uma origem confiável do servidor, sem ser aceito como campo do formulário de cadastro. A autenticação ainda não foi implementada.

Os testes de serviço verificam normalização, bloqueio de nome vazio antes da persistência, propagação de contexto e conta e preservação de erros. Para executar também o teste de integração, use um banco exclusivo de teste com as migrações aplicadas e forneça `TEST_DATABASE_URL` ao executar `go test ./...` em `apps/api`. Esse teste verifica gravação real, vínculo com a conta, IDs distintos para nomes iguais, estado ativo inicial e cancelamento. Sem essa variável, a integração é explicitamente pulada; os demais testes continuam executando. A fixture cria sua própria conta e remove somente seus registros ao terminar.

## sequência sugerida

| etapa | entrega verificável |
| --- | --- |
| 1. base técnica | Protótipo com uma página React chamando uma rota Go sob `/api` no mesmo preview da Vercel; desenvolvimento local equivalente, banco, migrações e autenticação. |
| 2. primeiro fluxo completo | Cadastrar funcionário e tarifa, registrar caixas, ver resumo semanal e registrar pagamento. Validar os exemplos de cálculo dos [critérios de aceite](criterios-de-aceite.md). |
| 3. vendas e despesas | Registrar venda por classificação, desconto e recebimento; registrar energia e demais despesas; mostrar painel com vendido, recebido, devido, pago e saldo de caixa separados. |
| 4. insumos e calendário | Registrar produto, compra, estoque e pulverização; permitir cadastro rápido; mostrar calendário e mudança das fases da Lua. |
| 5. uso real | Observar o registro de uma semana com o cliente, corrigir rótulos e passos difíceis e conferir resultados com os controles que ele já usa. |

O primeiro fluxo completo é o **MVP de validação**: já resolve a rotina de maior frequência e permite descobrir cedo se o lançamento por funcionário e o fechamento semanal estão simples o bastante.

## contrato inicial da api

Manter rotas versionadas sob `/api/v1` e respostas JSON. Uma proposta mínima para o primeiro fluxo é:

```text
GET/POST       /api/v1/employees
PATCH          /api/v1/employees/{id}       # renomear
DELETE         /api/v1/employees/{id}       # desativar, sem apagar histórico
GET/POST       /api/v1/harvest-rates
GET/POST       /api/v1/harvest-entries
PATCH          /api/v1/harvest-entries/{id}
GET            /api/v1/payroll-weeks/{week}/summary
POST           /api/v1/employee-payments
```

O servidor calcula totais e saldos; o frontend apresenta o resultado e não é a fonte de verdade dos valores. As operações de escrita precisam de validação, autorização, transação e proteção contra envio duplicado. Detalhar os demais endpoints quando as primeiras telas e regras estiverem validadas.

## verificações que importam

- Testes de domínio para tarifa por vigência, caixas normais e de moagem, diárias, pagamento parcial e saldo semanal.
- Testes de integração para uma correção após pagamento, venda com desconto, recebimento parcial e compra que afeta despesa e estoque uma só vez.
- Teste de segurança que inspeciona um banco de teste e confirma que valores monetários persistidos não aparecem em texto claro, além de teste de restauração com as chaves corretas.
- Teste no celular de uma sequência real: registrar caixas para vários funcionários, fechar a semana e registrar uma pulverização com insumo novo.

## decisões técnicas antes do código de produção

- Definir modelo de criptografia por campo, serviço de chaves separado do banco e recuperação/rotação. Indicadores financeiros devem ser calculados no servidor a partir de dados autorizados; não persistir agregados sensíveis em texto aberto. [Referência: OWASP](https://cheatsheetseries.owasp.org/cheatsheets/Cryptographic_Storage_Cheat_Sheet.html).
- Validar o caminho de deploy de React estático e API Go em um projeto Vercel Services; a configuração final depende do protótipo e da documentação vigente, pois o recurso está em beta.
- Definir o calendário semanal, as unidades de estoque e a origem dos dados lunares com o cliente antes de fechar essas regras.
