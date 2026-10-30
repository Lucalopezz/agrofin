-- +goose Up
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    login_name TEXT NOT NULL UNIQUE CHECK (btrim(login_name) <> ''),
    password_hash TEXT NOT NULL CHECK (btrim(password_hash) <> ''),
    timezone TEXT NOT NULL DEFAULT 'America/Sao_Paulo',
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE employees (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    name TEXT NOT NULL CHECK (btrim(name) <> ''),
    active BOOLEAN NOT NULL DEFAULT true,
    deactivated_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, id),
    CHECK ((active AND deactivated_at IS NULL) OR (NOT active AND deactivated_at IS NOT NULL))
);

-- Nomes iguais são permitidos: o ID distingue pessoas e preserva o histórico.
CREATE INDEX employees_active_by_user_idx ON employees (user_id, active, name);

-- +goose Down
DROP TABLE employees;
DROP TABLE users;
