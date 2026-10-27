-- +goose Up
CREATE TABLE accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL CHECK (btrim(name) <> ''),
    timezone TEXT NOT NULL DEFAULT 'America/Sao_Paulo',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL UNIQUE REFERENCES accounts(id),
    login_name TEXT NOT NULL UNIQUE CHECK (btrim(login_name) <> ''),
    password_hash TEXT NOT NULL CHECK (btrim(password_hash) <> ''),
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- A chave composta permite vincular o autor ao mesmo cliente do lançamento.
    UNIQUE (account_id, id)
);

CREATE TABLE employees (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES accounts(id),
    name TEXT NOT NULL CHECK (btrim(name) <> ''),
    active BOOLEAN NOT NULL DEFAULT true,
    deactivated_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (account_id, id),
    CHECK ((active AND deactivated_at IS NULL) OR (NOT active AND deactivated_at IS NOT NULL))
);

-- Nomes iguais são permitidos: o ID distingue pessoas e preserva o histórico.
CREATE INDEX employees_active_by_account_idx ON employees (account_id, active, name);

-- +goose Down
DROP TABLE employees;
DROP TABLE users;
DROP TABLE accounts;
