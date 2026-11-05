# Linguagem de domínio

Este documento explica os principais nomes usados no código do Agrofin e como eles correspondem às tarefas reais da produção. Os identificadores do código usam inglês; a documentação e os textos apresentados ao cliente usam português.

## Colheita é um lançamento

`Harvest` representa um lançamento de colheita específico: uma pessoa, uma data e as caixas registradas naquele evento. Não é o funcionário nem todo o histórico dele.

O exemplo abaixo é ilustrativo. `data` representa uma data validada pelo sistema; o trecho não precisa compilar sozinho.

```go
Harvest{
    EmployeeID: "id-do-joao",
    HarvestDate: data,
    Items: []HarvestItem{
        {BoxType: "normal", Quantity: 10, AppliedRateCents: 500},
        {BoxType: "moagem", Quantity: 2, AppliedRateCents: 300},
    },
}
```

Na prática: João colheu 10 caixas normais e 2 de moagem naquele dia. Cada `HarvestItem` guarda um tipo, a quantidade e a tarifa unitária que valeu naquele lançamento. O funcionário continua sendo um cadastro separado, ligado por `EmployeeID`; assim, renomeá-lo não exige carregar ou regravar suas colheitas.

O lançamento é a raiz do agregado: seus itens pertencem a ele e são persistidos juntos. Alterar o cadastro do funcionário não significa alterar os lançamentos anteriores. Uma consulta pode combinar os dois para exibir uma tela, sem transformar o histórico em parte do cadastro.

## Termos do código

| termo | significado no Agrofin | estado |
| --- | --- | --- |
| `Employee` | Funcionário cadastrado, sem login, vinculado ao cliente que gerencia a produção. | Cadastro disponível na API. |
| `Rate` | Tarifa unitária por tipo de caixa e vigência. | Criação e consulta disponíveis; valores protegidos no banco. |
| `HarvestItem` | Linha de um lançamento: tipo de caixa, quantidade e tarifa aplicada. | Disponível dentro do lançamento. |
| `Harvest` | Um evento de colheita de um funcionário em uma data, com um ou mais itens. | Criação e consulta por período disponíveis. |
| `HarvestPeriod` | Intervalo de datas usado para consultar lançamentos; as duas datas são inclusivas. | Usado na consulta da API. |
| `ProtectedHarvest` | Forma interna de persistência do lançamento, com os valores financeiros cifrados. | Usada pelo repositório PostgreSQL. |
| `DailyWork` | Uma diária lançada para um funcionário e uma data, separada das caixas. | Criação disponível em `POST /daily-work`; valor protegido no banco. |
| `ProtectedDailyWork` | Forma interna de persistência da diária, com centavos substituídos pelo envelope cifrado. | Recebida pelo repositório PostgreSQL. |
| `AuditEvent` | Registro técnico de uma alteração com dados protegidos. | A tabela existe no esquema, mas a V1 não registra auditoria de edição de colheitas. |
| `Summary` | Projeção de leitura que combina lançamentos para mostrar totais em uma tela. Não é um novo cadastro. | Resumo inicial por funcionário e intervalo disponível em `GET /harvest-entries/summary`; não representa um agregado persistido. |

## Por que as caixas não ficam dentro de `Employee`

Um funcionário pode ter muitos lançamentos ao longo do tempo, em datas e tipos de caixa diferentes. Cada lançamento também guarda a tarifa aplicada naquele dia. Se o histórico fizesse parte do cadastro, uma operação simples — como renomear alguém — acabaria carregando um conjunto grande de dados que tem outra finalidade e outro ciclo de vida.

Por isso, `Harvest` referencia o funcionário pelo ID. Para uma tela de resumo, o serviço pode consultar os lançamentos de um período, agrupar por funcionário e devolver um DTO de leitura com nome, quantidades e valores. Esse resultado atende à tela sem criar contadores editáveis no funcionário nem mudar a forma de persistência do histórico.

## Classificações e dinheiro

`BoxType` classifica caixas para calcular a remuneração da colheita: `normal` ou `moagem`. A classificação da venda ao barracão é outro conceito (`exportação`, `mercado` ou `moagem`) e não deve ser usada como tipo de caixa da colheita.

`AppliedRateCents` representa centavos inteiros em memória ou na resposta autorizada. No banco, o valor correspondente fica cifrado dentro de `ProtectedHarvest`; a tarifa aplicada é uma fotografia do lançamento e não muda quando uma tarifa futura é cadastrada.

`DailyWork.AmountCents` também representa centavos inteiros. A diária é um lançamento financeiro próprio e não se converte em caixas nem aparece no resumo inicial de colheitas. `ProtectedDailyWork` leva ao repositório somente ciphertext, nonce e versão da chave para esse valor.
