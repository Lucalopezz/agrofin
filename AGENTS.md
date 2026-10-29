# orientações para agentes

## contexto e estado atual

Leia [CONTEXT.md](CONTEXT.md) antes de planejar alterações. Este repositório contém documentação de produto, API Go mínima com conexão ao PostgreSQL e migrações do esquema; a rota de funcionários ainda devolve uma listagem vazia e o frontend React possui uma base com Vite, Tailwind e shadcn, sem telas de negócio. Trate os demais fluxos dos documentos como proposta até que o código e o deploy sejam validados.

Use [histórias de usuário](docs/historias-de-usuario.md), [requisitos](docs/requisitos.md) e [critérios de aceite](docs/criterios-de-aceite.md) para entender o comportamento. Consulte o [modelo de dados](docs/modelo-de-dados.md), os [fluxos mobile](docs/fluxos-mobile.md) e o [plano de implementação](docs/plano-de-implementacao.md) conforme a tarefa. Não transforme uma decisão pendente em regra definitiva sem evidência ou resposta do usuário.

## escolhas do projeto

- Produto para gestão de produção de limão, usado principalmente em celular por uma pessoa que precisa de telas simples e poucos passos.
- Monorepo planejado com `apps/api` em Go e `apps/web` em React + TypeScript. No Go, seguir convenções da linguagem, com entrada em `cmd` e código interno em `internal` quando fizer sentido. No React, organizar código de domínio por `src/features/<feature>`.
- Separar a classificação usada para pagar a colheita (normal e moagem) da classificação de venda ao barracão (exportação, mercado e moagem).
- O cadastro de funcionários recebe apenas nome; o identificador e o estado ativo/inativo são técnicos. Tarifas e pagamentos pertencem a fluxos separados. O pacote `internal/funcionarios` é dono desse cadastro e de suas rotas.
- Só o cliente, gerente da plantação, usa o sistema. Funcionários não possuem login. A equipe de cada semana resulta dos lançamentos daquela semana; não criar cadastro semanal nem desativar automaticamente quem não trabalhou em uma semana.
- Ausência de lançamento em um dia não representa falta nem gera penalidade.
- Valores financeiros exigem proteção contra vazamento isolado do banco. Não introduzir persistência em texto claro de tarifas, preços, pagamentos, recebimentos ou despesas. Consultar a skill de dados financeiros antes de alterar essa área.
- Em textos de produto e documentação, usar **cliente** para o administrador da produção. Não registrar relação familiar ou dados pessoais do caso que motivou o projeto.
- Documentação e interface em português, acompanhando o material atual. Identificadores de código podem usar inglês. Manter títulos em minúsculas.

## skills locais

As três skills `agrofin-*` foram adaptadas para este projeto a partir dos fluxos de estrutura, domínio e frontend encontrados no DevLog. Leia apenas as que correspondem ao trabalho atual.

| skill | quando usar |
| --- | --- |
| [agrofin-go-domain](.agents/skills/agrofin-go-domain/SKILL.md) | Regras de colheita, pagamentos, vendas, estoque ou API Go. |
| [agrofin-react-mobile](.agents/skills/agrofin-react-mobile/SKILL.md) | Telas, formulários, dados remotos e interações React no celular. |
| [agrofin-financial-data](.agents/skills/agrofin-financial-data/SKILL.md) | Persistência, relatórios, autenticação ou criptografia de dados financeiros. |

## skills externas versionadas no repositório

Estas skills foram instaladas de fontes públicas, com origem e revisão registradas em [skills externas](docs/skills-externas.md). Use somente as aplicáveis; as escolhas do projeto acima e a documentação vigente prevalecem sobre exemplos genéricos das skills.

| skill | quando usar |
| --- | --- |
| [react-best-practices](.agents/skills/react-best-practices/SKILL.md) | Escrever ou revisar React; aplicar as regras de React pertinentes ao projeto, sem assumir Next.js. |
| [web-design-guidelines](.agents/skills/web-design-guidelines/SKILL.md) | Revisar acessibilidade, usabilidade e apresentação das telas. |
| [golang-context](.agents/skills/golang-context/SKILL.md) | Propagação de cancelamento e prazo entre handler, domínio e banco Go. |
| [golang-naming](.agents/skills/golang-naming/SKILL.md) | Nomear pacotes, tipos, funções e erros Go. |
| [security-best-practices](.agents/skills/security-best-practices/SKILL.md) | Revisão de segurança de Go e React ou implementação segura quando a tarefa pedir isso. |

## skills da vercel já disponíveis no ambiente

Estas skills vêm do plugin da Vercel e não foram copiadas para o repositório. Use a skill aplicável e confira a documentação atual antes de configurar serviços ou fazer deploy:

- `vercel:vercel-services` para o deploy unificado de web e API; o recurso está em beta.
- `vercel:vercel-functions` para a execução da API Go.
- `vercel:vercel-storage` e `vercel:env-vars` para banco e configuração.
- `vercel:deployments-cicd` para preview, produção e CI.
- `vercel:verification` para verificar o fluxo completo após iniciar o app.

Para React, a versão oficial da Vercel instalada acima fica registrada no repositório. As skills do plugin continuam úteis para recursos específicos da plataforma e devem ser verificadas contra a documentação atual.

Não configurar deploy só porque uma skill existe. O primeiro passo de implementação previsto é validar com um protótipo mínimo que React e Go funcionem no mesmo projeto Vercel.

## trabalho no repositório

- Antes de criar arquivos, identifique a aplicação e o domínio que os possuem. Código compartilhado precisa ter uso real em mais de um lugar.
- Na API Go, organize arquivos por responsabilidade concreta dentro do pacote do domínio: `handler.go` coordena HTTP, `handler_request.go` trata entrada e validação do protocolo e `handler_response.go` define respostas e traduz erros para HTTP. Mantenha o service independente de HTTP; funções auxiliares sem estado do `Handler` são funções privadas do pacote. Só crie esses arquivos quando houver conteúdo real, sem camadas vazias ou pacote genérico `helpers`.
- Ao alterar uma regra de negócio, verifique o exemplo correspondente nos critérios de aceite e proteja o comportamento com o menor teste que capture o risco.
- Explique decisões importantes de Go e seus trade-offs: este também é um projeto de estudo.
- Preserve alterações já feitas pelo usuário. Atualize a documentação quando o comportamento implementado divergir do plano e distinga claramente proposta de funcionalidade entregue.
- Ao criar commits, use Conventional Commits com mensagem em pt-BR, como `feat:`, `fix:` ou `refact:`.

## comentários no código

- Sempre adicione comentários próximos de toda lógica não óbvia e das escolhas cuja razão não seja evidente para outro desenvolvedor. Explique **por que** o código foi feito daquela forma, a regra ou invariante que protege e os trade-offs relevantes.
- Aplique a regra em Go, React, testes e configurações quando houver comportamento que precise de contexto. Escreva comentários em português e mantenha-os atualizados quando o código mudar.
- Não use comentários que apenas repitam a instrução visível na linha; eles devem acrescentar o contexto necessário para entender e manter o código.
