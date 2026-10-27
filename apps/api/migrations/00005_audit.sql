-- +goose Up
CREATE TABLE audit_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES accounts(id),
    actor_id UUID NOT NULL,
    happened_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    entity_type TEXT NOT NULL CHECK (btrim(entity_type) <> ''),
    entity_id UUID NOT NULL,
    operation TEXT NOT NULL CHECK (btrim(operation) <> ''),
    -- Antes/depois ficam em um único envelope cifrado para impedir vazamento
    -- de valores financeiros por uma cópia isolada do banco de auditoria.
    payload_ciphertext BYTEA NOT NULL CHECK (octet_length(payload_ciphertext) > 0),
    payload_nonce BYTEA NOT NULL CHECK (octet_length(payload_nonce) > 0),
    payload_key_version TEXT NOT NULL CHECK (btrim(payload_key_version) <> ''),
    FOREIGN KEY (account_id, actor_id) REFERENCES users (account_id, id)
);

CREATE INDEX audit_events_entity_idx ON audit_events (account_id, entity_type, entity_id, happened_at);

-- +goose Down
DROP TABLE audit_events;
