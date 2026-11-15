-- +goose Up
-- Preserva chaves de registros existentes, mas novos lançamentos não exigem chave.
ALTER TABLE expenses ALTER COLUMN request_key DROP NOT NULL;

-- +goose Down
-- O rollback exige resolver previamente despesas criadas sem chave, sem inventar dados.
ALTER TABLE expenses ALTER COLUMN request_key SET NOT NULL;
