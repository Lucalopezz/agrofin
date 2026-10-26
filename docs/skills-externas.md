# skills externas

Estas skills foram baixadas de repositórios públicos para `.agents/skills/`, de modo que o projeto tenha uma versão conhecida das orientações usadas pelos agentes. Elas complementam as skills específicas do agrofin; não substituem seus requisitos ou as instruções do usuário.

| skill | origem | revisão | uso neste projeto |
| --- | --- | --- | --- |
| `react-best-practices` | [Vercel](https://github.com/vercel-labs/agent-skills/tree/063bee94c3f4df8453406c830b0a7df0f2860278/skills/react-best-practices) | `063bee94c3f4df8453406c830b0a7df0f2860278` | Práticas de React aplicáveis ao frontend planejado. |
| `web-design-guidelines` | [Vercel](https://github.com/vercel-labs/agent-skills/tree/063bee94c3f4df8453406c830b0a7df0f2860278/skills/web-design-guidelines) | `063bee94c3f4df8453406c830b0a7df0f2860278` | Revisão de interface e acessibilidade. |
| `golang-context` | [samber](https://github.com/samber/cc-skills-golang/tree/246b6c4c1ab3077dee114beb9ade3bb0527dfebc/skills/golang-context) | `246b6c4c1ab3077dee114beb9ade3bb0527dfebc` | Uso idiomático de `context.Context` na API. |
| `golang-naming` | [samber](https://github.com/samber/cc-skills-golang/tree/246b6c4c1ab3077dee114beb9ade3bb0527dfebc/skills/golang-naming) | `246b6c4c1ab3077dee114beb9ade3bb0527dfebc` | Convenções de nomes em Go. |
| `security-best-practices` | [OpenAI](https://github.com/openai/skills/tree/49f948faa9258a0c61caceaf225e179651397431/skills/.curated/security-best-practices) | `49f948faa9258a0c61caceaf225e179651397431` | Orientação de segurança para Go e React quando a tarefa envolver segurança. |

Os diretórios da Vercel e da OpenAI foram copiados sem alteração no conteúdo das skills. Nas duas skills Go, removemos apenas os campos de frontmatter `user-invocable`, `compatibility` e `paths`, que o validador de skills deste ambiente não aceita; o texto de orientação permanece igual ao upstream. Cada skill Go inclui a licença MIT do repositório de origem. A skill de segurança inclui seu próprio `LICENSE.txt`.

A skill `web-design-guidelines` busca as diretrizes atuais da Vercel a cada revisão; a revisão fixada acima identifica o arquivo da skill, mas o conteúdo consultado por ela pode mudar.

As skills da Vercel para Services, Functions, armazenamento e deploy já são fornecidas pelo plugin do ambiente. Elas não foram duplicadas aqui, pois tratam de uma plataforma que muda com frequência. Para atualizar uma skill versionada, compare o novo upstream com a revisão acima antes de substituir os arquivos e registre a nova revisão neste documento.
