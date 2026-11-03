package colheitas

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository persiste e lê tarifas gerais usando o pool compartilhado da aplicação.
// Na escrita, recebe os metadados validados e o valor já protegido pelo Service.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository não abre nem fecha conexões; quem criou o pool mantém seu ciclo de vida.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// HasRateOverlap consulta apenas o gerente, tipo e período validados pelo serviço.
func (r *Repository) HasRateOverlap(ctx context.Context, period RatePeriod) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1
			FROM harvest_rates
			WHERE user_id = $1
			  AND employee_id IS NULL
			  AND box_type = $2
			  AND daterange(valid_from, valid_until, '[)')
			      && daterange($3::date, $4::date, '[)')
		)
	`
	// A mesma representação da restrição do banco mantém início inclusivo e fim
	// exclusivo. Um fim NULL abre o período; employee_id NULL restringe ao escopo geral.
	var overlaps bool
	err := r.pool.QueryRow(ctx, query, period.UserID, string(period.BoxType), period.ValidFrom, period.ValidUntil).Scan(&overlaps)
	if err != nil {
		return false, fmt.Errorf("consultar sobreposição de tarifa: %w", err)
	}
	return overlaps, nil
}

// NextRateID gera o UUID no PostgreSQL antes da cifra, sem inserir um registro parcial.
func (r *Repository) NextRateID(ctx context.Context) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx, "SELECT gen_random_uuid()::text").Scan(&id)
	if err != nil {
		return "", fmt.Errorf("gerar identificador da tarifa: %w", err)
	}
	return id, nil
}

// CreateRate grava a entidade protegida sem normalizar dados ou cifrar valores.
func (r *Repository) CreateRate(ctx context.Context, rate ProtectedRate) error {
	const query = `
		INSERT INTO harvest_rates (
			id, user_id, employee_id, box_type,
			rate_ciphertext, rate_nonce, rate_key_version, valid_from, valid_until
		)
		VALUES ($1, $2, NULL, $3, $4, $5, $6, $7, $8)
	`
	// Uma única instrução persiste metadados e envelope juntos. NULL fixa a tarifa
	// geral; a entidade de entrada não permite escolher um funcionário individual.
	_, err := r.pool.Exec(ctx, query,
		rate.ID, rate.UserID, string(rate.BoxType), rate.Amount.Ciphertext,
		rate.Amount.Nonce, rate.Amount.KeyVersion, rate.ValidFrom, rate.ValidUntil,
	)
	if err != nil {
		var databaseError *pgconn.PgError
		if errors.As(err, &databaseError) && databaseError.Code == "23P01" && databaseError.ConstraintName == "harvest_rates_general_no_overlap" {
			// Outra requisição pode gravar depois da consulta do serviço. Só esta
			// exclusão representa sobreposição; outros erros mantêm sua identidade.
			return ErrRateOverlap
		}
		return fmt.Errorf("criar tarifa protegida: %w", err)
	}
	return nil
}

// ListRates lê somente tarifas gerais da conta, mantendo os valores cifrados
// até que o Service autentique e descriptografe cada envelope.
func (r *Repository) ListRates(ctx context.Context, userID string) ([]ProtectedRate, error) {
	const query = `
		SELECT id::text, box_type,
		       rate_ciphertext, rate_nonce, rate_key_version,
		       valid_from, valid_until
		FROM harvest_rates
		WHERE user_id = $1
		  AND employee_id IS NULL
		ORDER BY valid_from DESC, box_type ASC, id ASC
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("listar tarifas protegidas: %w", err)
	}
	defer rows.Close()

	rates := make([]ProtectedRate, 0)
	for rows.Next() {
		var rate ProtectedRate
		var boxType string
		err := rows.Scan(
			&rate.ID, &boxType,
			&rate.Amount.Ciphertext, &rate.Amount.Nonce, &rate.Amount.KeyVersion,
			&rate.ValidFrom, &rate.ValidUntil,
		)
		if err != nil {
			return nil, fmt.Errorf("ler tarifa protegida: %w", err)
		}
		// O filtro do SQL usa o dono recebido do Service; o mesmo valor confiável
		// será usado depois pela cifra, sem depender de user_id vindo da linha.
		rate.UserID = userID
		rate.BoxType = BoxType(boxType)
		rates = append(rates, rate)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("listar tarifas protegidas: %w", err)
	}
	return rates, nil
}
