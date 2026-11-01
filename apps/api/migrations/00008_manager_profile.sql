-- +goose Up
ALTER TABLE users ADD COLUMN name TEXT;

-- Cadastros anteriores não tinham nome de exibição. Usar temporariamente o login
-- preserva a conta existente até o cliente atualizar o próprio perfil.
UPDATE users SET name = login_name WHERE name IS NULL;

ALTER TABLE users ALTER COLUMN name SET NOT NULL;
ALTER TABLE users
    ADD CONSTRAINT users_name_length_check
    CHECK (char_length(btrim(name)) BETWEEN 1 AND 100);

-- +goose Down
ALTER TABLE users DROP CONSTRAINT users_name_length_check;
ALTER TABLE users DROP COLUMN name;
