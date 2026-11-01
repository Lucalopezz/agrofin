-- +goose Up
-- Cada cliente pode criar sua conta; user_id continua delimitando os dados de
-- cada gerente. A unicidade de login_name é preservada pela restrição original.
DROP INDEX users_single_manager_idx;

-- +goose Down
-- Com várias contas, o rollback deve falhar em vez de apagar clientes para
-- restaurar a restrição antiga. Só pode ser aplicado com no máximo um usuário.
CREATE UNIQUE INDEX users_single_manager_idx ON users ((true));
