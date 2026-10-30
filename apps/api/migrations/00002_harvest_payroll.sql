-- +goose Up
-- O GiST combina igualdade de UUID/texto com interseção de períodos para evitar
-- duas tarifas simultâneas no mesmo escopo, inclusive sob gravações concorrentes.
CREATE EXTENSION IF NOT EXISTS btree_gist;

CREATE TABLE harvest_rates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    employee_id UUID,
    box_type TEXT NOT NULL CHECK (box_type IN ('normal', 'moagem')),
    rate_ciphertext BYTEA NOT NULL CHECK (octet_length(rate_ciphertext) > 0),
    rate_nonce BYTEA NOT NULL CHECK (octet_length(rate_nonce) > 0),
    rate_key_version TEXT NOT NULL CHECK (btrim(rate_key_version) <> ''),
    valid_from DATE NOT NULL,
    valid_until DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    FOREIGN KEY (user_id, employee_id) REFERENCES employees (user_id, id),
    CHECK (valid_until IS NULL OR valid_until > valid_from)
);

-- A vigência usa limite final exclusivo. Tarifa geral e individual podem
-- coexistir; qual prevalece entre elas ainda é uma decisão de negócio.
ALTER TABLE harvest_rates ADD CONSTRAINT harvest_rates_general_no_overlap
    EXCLUDE USING gist (
        user_id WITH =,
        box_type WITH =,
        daterange(valid_from, valid_until, '[)') WITH &&
    ) WHERE (employee_id IS NULL);

ALTER TABLE harvest_rates ADD CONSTRAINT harvest_rates_employee_no_overlap
    EXCLUDE USING gist (
        user_id WITH =,
        employee_id WITH =,
        box_type WITH =,
        daterange(valid_from, valid_until, '[)') WITH &&
    ) WHERE (employee_id IS NOT NULL);

CREATE TABLE harvests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    employee_id UUID NOT NULL,
    harvest_date DATE NOT NULL,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, id),
    FOREIGN KEY (user_id, employee_id) REFERENCES employees (user_id, id),
    FOREIGN KEY (created_by) REFERENCES users (id)
);

CREATE INDEX harvests_user_date_idx ON harvests (user_id, harvest_date, employee_id);

CREATE TABLE harvest_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    harvest_id UUID NOT NULL,
    box_type TEXT NOT NULL CHECK (box_type IN ('normal', 'moagem')),
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    -- A tarifa aplicada é uma fotografia cifrada, não uma referência à tarifa atual.
    applied_rate_ciphertext BYTEA NOT NULL CHECK (octet_length(applied_rate_ciphertext) > 0),
    applied_rate_nonce BYTEA NOT NULL CHECK (octet_length(applied_rate_nonce) > 0),
    applied_rate_key_version TEXT NOT NULL CHECK (btrim(applied_rate_key_version) <> ''),
    FOREIGN KEY (user_id, harvest_id) REFERENCES harvests (user_id, id),
    UNIQUE (harvest_id, box_type)
);

CREATE TABLE daily_work (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    employee_id UUID NOT NULL,
    work_date DATE NOT NULL,
    amount_ciphertext BYTEA NOT NULL CHECK (octet_length(amount_ciphertext) > 0),
    amount_nonce BYTEA NOT NULL CHECK (octet_length(amount_nonce) > 0),
    amount_key_version TEXT NOT NULL CHECK (btrim(amount_key_version) <> ''),
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    FOREIGN KEY (user_id, employee_id) REFERENCES employees (user_id, id)
);

CREATE INDEX daily_work_user_date_idx ON daily_work (user_id, work_date, employee_id);

CREATE TABLE employee_payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    employee_id UUID NOT NULL,
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,
    paid_at TIMESTAMPTZ NOT NULL,
    amount_ciphertext BYTEA NOT NULL CHECK (octet_length(amount_ciphertext) > 0),
    amount_nonce BYTEA NOT NULL CHECK (octet_length(amount_nonce) > 0),
    amount_key_version TEXT NOT NULL CHECK (btrim(amount_key_version) <> ''),
    notes TEXT,
    -- Uma repetição da mesma requisição não pode criar outra saída de caixa.
    request_key UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    FOREIGN KEY (user_id, employee_id) REFERENCES employees (user_id, id),
    UNIQUE (user_id, request_key),
    CHECK (period_end > period_start)
);

CREATE INDEX employee_payments_period_idx ON employee_payments (user_id, employee_id, period_start, period_end);

CREATE TABLE payroll_adjustments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    employee_id UUID NOT NULL,
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,
    adjustment_type TEXT NOT NULL CHECK (adjustment_type IN ('credit', 'debit')),
    amount_ciphertext BYTEA NOT NULL CHECK (octet_length(amount_ciphertext) > 0),
    amount_nonce BYTEA NOT NULL CHECK (octet_length(amount_nonce) > 0),
    amount_key_version TEXT NOT NULL CHECK (btrim(amount_key_version) <> ''),
    reason TEXT NOT NULL CHECK (btrim(reason) <> ''),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    FOREIGN KEY (user_id, employee_id) REFERENCES employees (user_id, id),
    CHECK (period_end > period_start)
);

CREATE INDEX payroll_adjustments_period_idx ON payroll_adjustments (user_id, employee_id, period_start, period_end);

-- +goose Down
DROP TABLE payroll_adjustments;
DROP TABLE employee_payments;
DROP TABLE daily_work;
DROP TABLE harvest_items;
DROP TABLE harvests;
DROP TABLE harvest_rates;
