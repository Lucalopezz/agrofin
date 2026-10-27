-- +goose Up
-- O produto confirmado tem um único gerente em toda a instalação. A tabela
-- separada existe para autenticação, mas não representa acesso de funcionários.
CREATE UNIQUE INDEX users_single_manager_idx ON users ((true));

-- +goose Down
DROP INDEX users_single_manager_idx;
