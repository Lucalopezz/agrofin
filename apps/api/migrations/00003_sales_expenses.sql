-- +goose Up
CREATE TABLE sales (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    sold_at DATE NOT NULL,
    buyer_name TEXT,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, id)
);

CREATE INDEX sales_user_date_idx ON sales (user_id, sold_at);

CREATE TABLE sale_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    sale_id UUID NOT NULL,
    sale_type TEXT NOT NULL CHECK (sale_type IN ('exportacao', 'mercado', 'moagem')),
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    unit_price_ciphertext BYTEA NOT NULL CHECK (octet_length(unit_price_ciphertext) > 0),
    unit_price_nonce BYTEA NOT NULL CHECK (octet_length(unit_price_nonce) > 0),
    unit_price_key_version TEXT NOT NULL CHECK (btrim(unit_price_key_version) <> ''),
    FOREIGN KEY (user_id, sale_id) REFERENCES sales (user_id, id)
);

CREATE TABLE sale_discounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    sale_id UUID NOT NULL,
    reason TEXT NOT NULL CHECK (btrim(reason) <> ''),
    amount_ciphertext BYTEA NOT NULL CHECK (octet_length(amount_ciphertext) > 0),
    amount_nonce BYTEA NOT NULL CHECK (octet_length(amount_nonce) > 0),
    amount_key_version TEXT NOT NULL CHECK (btrim(amount_key_version) <> ''),
    FOREIGN KEY (user_id, sale_id) REFERENCES sales (user_id, id)
);

CREATE TABLE sale_receipts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    sale_id UUID NOT NULL,
    received_at TIMESTAMPTZ NOT NULL,
    amount_ciphertext BYTEA NOT NULL CHECK (octet_length(amount_ciphertext) > 0),
    amount_nonce BYTEA NOT NULL CHECK (octet_length(amount_nonce) > 0),
    amount_key_version TEXT NOT NULL CHECK (btrim(amount_key_version) <> ''),
    request_key UUID NOT NULL,
    FOREIGN KEY (user_id, sale_id) REFERENCES sales (user_id, id),
    UNIQUE (user_id, request_key)
);

CREATE INDEX sale_receipts_user_date_idx ON sale_receipts (user_id, received_at);

-- Despesas gerais são saídas já pagas. Compras de insumos têm pagamentos próprios,
-- evitando registrar a mesma saída também em expenses.
CREATE TABLE expenses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    paid_at TIMESTAMPTZ NOT NULL,
    category TEXT NOT NULL CHECK (btrim(category) <> ''),
    description TEXT,
    amount_ciphertext BYTEA NOT NULL CHECK (octet_length(amount_ciphertext) > 0),
    amount_nonce BYTEA NOT NULL CHECK (octet_length(amount_nonce) > 0),
    amount_key_version TEXT NOT NULL CHECK (btrim(amount_key_version) <> ''),
    request_key UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, request_key)
);

CREATE INDEX expenses_user_date_idx ON expenses (user_id, paid_at);

-- +goose Down
DROP TABLE expenses;
DROP TABLE sale_receipts;
DROP TABLE sale_discounts;
DROP TABLE sale_items;
DROP TABLE sales;
