-- +goose Up
-- Cada login cria uma sessão opaca. O navegador recebe o token aleatório, mas
-- este banco guarda só SHA-256, para que um dump não seja suficiente para entrar.
CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- A sessão pertence ao gerente; apagar o usuário deve remover suas sessões.
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    -- SHA-256 tem 32 bytes; UNIQUE impede colisão/reuso de token entre sessões.
    token_hash BYTEA NOT NULL UNIQUE CHECK (octet_length(token_hash) = 32),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- Atualizado quando a sessão é validada; permite expiração por inatividade.
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- Limite absoluto, independente de atividade contínua.
    expires_at TIMESTAMPTZ NOT NULL,
    -- Revogar sem depender de o navegador ter apagado corretamente o cookie.
    revoked_at TIMESTAMPTZ,
    CHECK (expires_at > created_at)
);

-- Ajuda a consultar sessões ativas de um usuário sem incluir linhas revogadas.
CREATE INDEX sessions_active_by_user_idx ON sessions (user_id, expires_at)
WHERE revoked_at IS NULL;

-- +goose Down
DROP TABLE sessions;
