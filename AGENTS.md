# orientações para agentes

## contexto e estado atual

Leia [CONTEXT.md](CONTEXT.md) antes de planejar alterações. Este repositório contém documentação de produto e planejamento; ainda não há aplicações Go ou React implementadas. Trate a arquitetura dos documentos como proposta até que o código e o deploy sejam validados.

Use [histórias de usuário](docs/historias-de-usuario.md), [requisitos](docs/requisitos.md) e [critérios de aceite](docs/criterios-de-aceite.md) para entender o comportamento. Consulte o [modelo de dados](docs/modelo-de-dados.md), os [fluxos mobile](docs/fluxos-mobile.md) e o [plano de implementação](docs/plano-de-implementacao.md) conforme a tarefa. Não transforme uma decisão pendente em regra definitiva sem evidência ou resposta do usuário.

## escolhas do projeto

- Produto para gestão de produção de limão, usado principalmente em celular por uma pessoa que precisa de telas simples e poucos passos.
- Monorepo planejado com `apps/api` em Go e `apps/web` em React + TypeScript. No Go, seguir convenções da linguagem, com entrada em `cmd` e código interno em `internal` quando fizer sentido. No React, organizar código de domínio por `src/features/<feature>`.
- Separar a classificação usada para pagar a colheita (normal e moagem) da classificação de venda ao barracão (exportação, mercado e moagem).
- Ausência de lançamento em um dia não representa falta nem gera penalidade.
- Valores financeiros exigem proteção contra vazamento isolado do banco. Não introduzir persistência em texto claro de tarifas, preços, pagamentos, recebimentos ou despesas. Consultar a skill de dados financeiros antes de alterar essa área.
- Em textos de produto e documentação, usar **cliente** para o administrador da produção. Não registrar relação familiar ou dados pessoais do caso que motivou o projeto.
- Documentação e interface em português, acompanhando o material atual. Identificadores de código podem usar inglês. Manter títulos em minúsculas.

## skills locais

As skills em `.agents/skills/` foram adaptadas para este projeto a partir dos fluxos de estrutura, domínio e frontend encontrados no DevLog. Leia apenas as que correspondem ao trabalho atual.

| skill | quando usar |
| --- | --- |
| [agrofin-go-domain](.agents/skills/agrofin-go-domain/SKILL.md) | Regras de colheita, pagamentos, vendas, estoque ou API Go. |
| [agrofin-react-mobile](.agents/skills/agrofin-react-mobile/SKILL.md) | Telas, formulários, dados remotos e interações React no celular. |
| [agrofin-financial-data](.agents/skills/agrofin-financial-data/SKILL.md) | Persistência, relatórios, autenticação ou criptografia de dados financeiros. |

## skills da vercel já disponíveis no ambiente

Estas skills vêm do plugin da Vercel e não foram copiadas para o repositório. Use a skill aplicável e confira a documentação atual antes de configurar serviços ou fazer deploy:

- `vercel:vercel-services` para o deploy unificado de web e API; o recurso está em beta.
- `vercel:vercel-functions` para a execução da API Go.
- `vercel:vercel-storage` e `vercel:env-vars` para banco e configuração.
- `vercel:deployments-cicd` para preview, produção e CI.
- `vercel:verification` para verificar o fluxo completo após iniciar o app.
- `vercel:react-best-practices` depois de alterações em vários componentes TSX.

Não configurar deploy só porque uma skill existe. O primeiro passo de implementação previsto é validar com um protótipo mínimo que React e Go funcionem no mesmo projeto Vercel.

## trabalho no repositório

- Antes de criar arquivos, identifique a aplicação e o domínio que os possuem. Código compartilhado precisa ter uso real em mais de um lugar.
- Ao alterar uma regra de negócio, verifique o exemplo correspondente nos critérios de aceite e proteja o comportamento com o menor teste que capture o risco.
- Explique decisões importantes de Go e seus trade-offs: este também é um projeto de estudo. Prefira comentários curtos junto à lógica não óbvia.
- Preserve alterações já feitas pelo usuário. Atualize a documentação quando o comportamento implementado divergir do plano e distinga claramente proposta de funcionalidade entregue.
