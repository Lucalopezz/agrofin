-- +goose Up
CREATE TABLE input_products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES accounts(id),
    name TEXT NOT NULL CHECK (btrim(name) <> ''),
    category TEXT NOT NULL CHECK (category IN ('fertilizante', 'defensivo')),
    default_unit TEXT NOT NULL CHECK (btrim(default_unit) <> ''),
    -- Pendente permite cadastrar o produto mesmo sem conhecer o saldo inicial.
    stock_control_state TEXT NOT NULL DEFAULT 'pending' CHECK (stock_control_state IN ('pending', 'tracked')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (account_id, id)
);

CREATE TABLE input_purchases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES accounts(id),
    purchased_at DATE NOT NULL,
    supplier_name TEXT,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (account_id, id)
);

CREATE INDEX input_purchases_account_date_idx ON input_purchases (account_id, purchased_at);

CREATE TABLE purchase_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES accounts(id),
    purchase_id UUID NOT NULL,
    product_id UUID NOT NULL,
    quantity NUMERIC(18, 6) NOT NULL CHECK (quantity > 0),
    unit TEXT NOT NULL CHECK (btrim(unit) <> ''),
    unit_cost_ciphertext BYTEA NOT NULL CHECK (octet_length(unit_cost_ciphertext) > 0),
    unit_cost_nonce BYTEA NOT NULL CHECK (octet_length(unit_cost_nonce) > 0),
    unit_cost_key_version TEXT NOT NULL CHECK (btrim(unit_cost_key_version) <> ''),
    FOREIGN KEY (account_id, purchase_id) REFERENCES input_purchases (account_id, id),
    FOREIGN KEY (account_id, product_id) REFERENCES input_products (account_id, id),
    UNIQUE (account_id, product_id, id)
);

CREATE TABLE input_purchase_payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES accounts(id),
    purchase_id UUID NOT NULL,
    paid_at TIMESTAMPTZ NOT NULL,
    amount_ciphertext BYTEA NOT NULL CHECK (octet_length(amount_ciphertext) > 0),
    amount_nonce BYTEA NOT NULL CHECK (octet_length(amount_nonce) > 0),
    amount_key_version TEXT NOT NULL CHECK (btrim(amount_key_version) <> ''),
    request_key UUID NOT NULL,
    FOREIGN KEY (account_id, purchase_id) REFERENCES input_purchases (account_id, id),
    UNIQUE (account_id, request_key)
);

CREATE INDEX input_purchase_payments_account_date_idx ON input_purchase_payments (account_id, paid_at);

CREATE TABLE spray_applications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES accounts(id),
    applied_at DATE NOT NULL,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (account_id, id)
);

CREATE INDEX spray_applications_account_date_idx ON spray_applications (account_id, applied_at);

CREATE TABLE spraying_inputs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES accounts(id),
    spray_application_id UUID NOT NULL,
    product_id UUID NOT NULL,
    quantity NUMERIC(18, 6) NOT NULL CHECK (quantity > 0),
    unit TEXT NOT NULL CHECK (btrim(unit) <> ''),
    FOREIGN KEY (account_id, spray_application_id) REFERENCES spray_applications (account_id, id),
    FOREIGN KEY (account_id, product_id) REFERENCES input_products (account_id, id),
    UNIQUE (spray_application_id, product_id),
    UNIQUE (account_id, product_id, id)
);

CREATE TABLE stock_movements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES accounts(id),
    product_id UUID NOT NULL,
    movement_date DATE NOT NULL,
    quantity NUMERIC(18, 6) NOT NULL CHECK (quantity <> 0),
    unit TEXT NOT NULL CHECK (btrim(unit) <> ''),
    source_type TEXT NOT NULL CHECK (source_type IN ('opening', 'purchase', 'spraying', 'adjustment')),
    purchase_item_id UUID UNIQUE,
    spraying_input_id UUID UNIQUE,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    FOREIGN KEY (account_id, product_id) REFERENCES input_products (account_id, id),
    -- A FK também compara produto e conta: um movimento não pode apontar para
    -- a compra ou pulverização de outro insumo.
    FOREIGN KEY (account_id, product_id, purchase_item_id) REFERENCES purchase_items (account_id, product_id, id),
    FOREIGN KEY (account_id, product_id, spraying_input_id) REFERENCES spraying_inputs (account_id, product_id, id),
    CHECK (
        (source_type = 'purchase' AND purchase_item_id IS NOT NULL AND spraying_input_id IS NULL AND quantity > 0)
        OR (source_type = 'spraying' AND spraying_input_id IS NOT NULL AND purchase_item_id IS NULL AND quantity < 0)
        OR (source_type IN ('opening', 'adjustment') AND purchase_item_id IS NULL AND spraying_input_id IS NULL)
    )
);

CREATE INDEX stock_movements_balance_idx ON stock_movements (account_id, product_id, movement_date);

-- +goose Down
DROP TABLE stock_movements;
DROP TABLE spraying_inputs;
DROP TABLE spray_applications;
DROP TABLE input_purchase_payments;
DROP TABLE purchase_items;
DROP TABLE input_purchases;
DROP TABLE input_products;
