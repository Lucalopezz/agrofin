# modelo de dados inicial

Modelo conceitual para orientar a implementação. Nomes de tabelas e campos podem mudar após validar os fluxos com o cliente.

## entidades

| entidade | dados principais | observação |
| --- | --- | --- |
| `user` | identificação, credenciais, estado | inicialmente, uma conta administradora |
| `employee` | identificador, nome, estado ativo/inativo | apenas nome é informado no cadastro; desativação preserva histórico |
| `harvest_rate` | tipo de caixa, valor, início da vigência, funcionário opcional | definir se existe tarifa individual |
| `harvest_entry` | data, funcionário, tipo, quantidade, tarifa aplicada, total | a tarifa e o total são fotografados no lançamento |
| `daily_work` | data, funcionário, valor, observação | custo por diária, sem caixas fictícias |
| `employee_payment` | funcionário, semana de referência, data, valor, observação | múltiplos pagamentos podem quitar a mesma semana |
| `sale` | data, barracão, observação | cabeçalho de um acerto |
| `sale_item` | venda, classificação, quantidade, preço unitário, subtotal | exportação, mercado ou moagem |
| `sale_discount` | venda, motivo, valor | separa descontos do preço original |
| `sale_receipt` | venda, data, valor | registra dinheiro efetivamente recebido |
| `expense` | data de pagamento, categoria, valor, descrição, origem | inclui energia e compra de insumos pagos; evita duplicidade |
| `input_product` | nome, categoria, unidade, controle de estoque | fertilizante ou defensivo |
| `stock_movement` | insumo, data, tipo, quantidade, origem | compra, saldo inicial, uso ou ajuste |
| `spraying` | data, observações | evento do calendário |
| `spraying_input` | pulverização, insumo, quantidade usada | vincula insumo ao evento e à baixa de estoque |
| `audit_event` | autor, data, entidade, operação, antes/depois protegidos | trilha de correções relevantes |

## relações e invariantes

- Uma colheita pertence a um funcionário e usa **caixa normal** ou **caixa de moagem**. Uma venda tem uma ou mais linhas de **exportação**, **mercado** ou **moagem**. As duas classificações não são equivalentes.
- `harvest_entry` guarda a tarifa aplicada no momento do lançamento. Corrigir uma tarifa vigente não deve alterar silenciosamente registros antigos.
- `employee_payment` referencia a semana e o funcionário. O saldo semanal é o valor devido pelas colheitas e diárias daquela semana menos os pagamentos ligados a ela.
- Venda líquida = soma dos subtotais das linhas − soma dos descontos. Valor a receber = venda líquida − recebimentos. Ambos não devem ficar negativos por um lançamento comum.
- Compra de insumo cria despesa e entrada de estoque ligadas pela mesma origem. A pulverização cria saída de estoque, mas não cria uma segunda despesa pela compra já registrada.
- No desenho inicial, a data da despesa é a data do pagamento. Se houver compras a prazo, será necessário separar compra de pagamento antes de usar esses valores no saldo de caixa.
- Um insumo cadastrado durante a pulverização pode ter saldo inicial informado ou ficar com controle de estoque pendente; o evento de uso precisa continuar rastreável.
- Usar datas da operação no fuso `America/Sao_Paulo` por padrão; timestamps técnicos são armazenados em UTC. Confirmar o fuso antes de implementar.
- Valores monetários são representados como centavos inteiros na aplicação, evitando arredondamento de ponto flutuante. Persistir campos sensíveis cifrados, com versão da chave e metadados necessários à descriptografia.

## consequências da criptografia para relatórios

Como preços e valores ficarão cifrados no banco, consultas SQL comuns não poderão somá-los diretamente. O servidor deve buscar apenas os registros do período e da conta autorizada, descriptografar os campos necessários e calcular o resumo em memória. Para volumes maiores, avaliar agregados também cifrados e atualizados transacionalmente. Datas, tipos e quantidades podem ser indexados sem revelar valores monetários; ainda assim, esses metadados também merecem avaliação de privacidade.

As chaves ficam separadas do banco, com controle de acesso e plano de rotação. Essa separação protege o cenário de vazamento isolado do banco; uma invasão da aplicação com acesso às chaves exige outras defesas. [Referência: OWASP](https://cheatsheetseries.owasp.org/cheatsheets/Cryptographic_Storage_Cheat_Sheet.html).
