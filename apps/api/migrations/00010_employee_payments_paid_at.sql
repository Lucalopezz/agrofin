-- +goose Up
CREATE INDEX employee_payments_user_paid_at_idx ON employee_payments (user_id, paid_at, id);

-- +goose Down
DROP INDEX employee_payments_user_paid_at_idx;
