---
name: agrofin-react-mobile
description: Criar ou alterar telas e fluxos React + TypeScript do agrofin, especialmente lançamentos frequentes em celulares.
---

# interface react para celular

Leia [os fluxos mobile](../../../docs/fluxos-mobile.md) e os [critérios de aceite](../../../docs/criterios-de-aceite.md) da tarefa. Inspecione componentes e padrões existentes antes de criar outros.

## estrutura e estado

- Coloque requisições, tipos, hooks, formulários e componentes específicos em `src/features/<domínio>`. `shared/` recebe apenas código realmente independente do domínio. A composição de rotas fica em `app/`.
- Mantenha rascunhos de formulário separados dos dados confirmados pelo servidor. O servidor é a fonte dos valores devidos e recebidos; não substitua totais oficiais por cálculos locais silenciosos.
- Após salvar, atualize os resumos afetados. Se uma resposta de escrita se perder, não repita automaticamente a operação sem saber se foi concluída. Preserve o rascunho após falha recuperável.

## interação principal

- Priorize as ações de registrar caixas, pagar funcionário e registrar pulverização. Permita registrar vários funcionários mantendo a data, com confirmação clara do total antes de salvar.
- Use rótulos diretos, áreas de toque confortáveis, foco visível e erros junto aos campos. Confira a tela com teclado do celular aberto e sem rolagem horizontal. Não dependa só de cor ou ícone para comunicar estado.
- Mostre vendido, recebido, devido e pago com nomes diferentes. Um estado sem registros não deve parecer falha de conexão.
- O cadastro rápido de insumo deve devolver o usuário ao formulário de pulverização com o rascunho intacto.

Verifique no navegador os fluxos que envolvem navegação, foco, teclado e layout estreito. Use testes de componente para comportamento de formulário e atualização de dados quando houver risco concreto.
