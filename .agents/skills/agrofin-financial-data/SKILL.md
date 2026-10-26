---
name: agrofin-financial-data
description: Proteger e calcular valores financeiros do agrofin ao alterar persistência, relatórios, autenticação ou criptografia.
---

# dados financeiros

Leia [os requisitos de segurança](../../../docs/requisitos.md#requisitos-não-funcionais) e [o modelo de dados](../../../docs/modelo-de-dados.md). Confirme a implementação atual antes de alterar armazenamento ou relatórios. Consulte orientação criptográfica atual e oficial ao escolher algoritmos ou serviço de chaves.

- Defina explicitamente o cenário protegido: uma cópia isolada do banco não deve revelar tarifas, preços, descontos, pagamentos, recebimentos ou despesas. Metadados como datas e quantidades podem continuar visíveis; avalie essa exposição.
- Cifre valores financeiros na aplicação antes de persistir, com criptografia autenticada e identificação da versão da chave. Mantenha chaves separadas do banco e planeje rotação e recuperação. Senhas usam hash próprio para senhas.
- Autorize cada leitura e escrita no servidor. Nunca exponha valores sensíveis em logs, erros, telemetria ou resposta para usuário não autorizado.
- Cálculos e relatórios devem distinguir vendido de recebido, devido de pago e saldo de caixa de lucro. Se a cifra impedir soma SQL, calcule no servidor após buscar apenas o período e a conta autorizados.
- Teste o caso funcional, a confidencialidade do banco de teste e a recuperação de dados cifrados quando mudar o mecanismo de persistência ou as chaves.

Não escolha um provedor de banco ou chaves apenas porque aparece no plano; essa decisão segue aberta. Use [OWASP Cryptographic Storage](https://cheatsheetseries.owasp.org/cheatsheets/Cryptographic_Storage_Cheat_Sheet.html) como referência de segurança, além da documentação atual do provedor escolhido.
