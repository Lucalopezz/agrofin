-- +goose Up
-- Cadastros existentes podem continuar sem observações.
ALTER TABLE input_products ADD COLUMN notes TEXT;

-- +goose Down
ALTER TABLE input_products DROP COLUMN notes;
