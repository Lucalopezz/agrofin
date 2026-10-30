-- +goose Up
-- O produto confirmado tem um único gerente em toda a instalação. A tabela guarda a identidade do único gerente; funcionários não possuem acesso.
CREATE UNIQUE INDEX users_single_manager_idx ON users ((true));

-- +goose Down
DROP INDEX users_single_manager_idx;
