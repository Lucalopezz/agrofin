package colheitas

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	calendarDate "github.com/lucalopezz/agrofin/apps/api/internal/platform/date"
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

// CheckActiveEmployee dá erro de domínio antes de consultar tarifas. A verificação
// é repetida com bloqueio em CreateHarvest para continuar correta sob concorrência.
func (r *Repository) CheckActiveEmployee(ctx context.Context, userID, employeeID string) error {
	var active bool
	err := r.pool.QueryRow(ctx, `
		SELECT active
		FROM employees
		WHERE user_id = $1 AND id = $2
	`, userID, employeeID).Scan(&active)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrEmployeeNotFound
	}
	if err != nil {
		return fmt.Errorf("consultar funcionário da colheita: %w", err)
	}
	if !active {
		return ErrEmployeeInactive
	}
	return nil
}

// RateForDate lê apenas a tarifa geral vigente na data, mantendo o envelope
// cifrado até o serviço autenticá-lo com o dono da sessão e o ID da tarifa.
func (r *Repository) RateForDate(ctx context.Context, userID string, boxType BoxType, date time.Time) (ProtectedRate, error) {
	const query = `
		SELECT id::text, rate_ciphertext, rate_nonce, rate_key_version,
		       valid_from, valid_until
		FROM harvest_rates
		WHERE user_id = $1
		  AND employee_id IS NULL
		  AND box_type = $2
		  AND valid_from <= $3::date
		  AND (valid_until IS NULL OR valid_until > $3::date)
	`
	var rate ProtectedRate
	rate.UserID = userID
	rate.BoxType = boxType
	err := r.pool.QueryRow(ctx, query, userID, string(boxType), date).Scan(
		&rate.ID, &rate.Amount.Ciphertext, &rate.Amount.Nonce,
		&rate.Amount.KeyVersion, &rate.ValidFrom, &rate.ValidUntil,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ProtectedRate{}, ErrRateNotFound
	}
	if err != nil {
		return ProtectedRate{}, fmt.Errorf("consultar tarifa vigente: %w", err)
	}
	return rate, nil
}

// CreateHarvest grava o agregado em uma única transação. O bloqueio compartilhado
// mantém a leitura de active estável até que cabeçalho e itens estejam persistidos.
func (r *Repository) CreateHarvest(ctx context.Context, harvest ProtectedHarvest) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("iniciar gravação da colheita: %w", err)
	}
	defer func() {
		// Mesmo se a conexão HTTP for cancelada, tentamos liberar a transação aberta.
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		_ = tx.Rollback(cleanupCtx)
	}()

	var active bool
	err = tx.QueryRow(ctx, `
		SELECT active
		FROM employees
		WHERE user_id = $1 AND id = $2
		FOR SHARE
	`, harvest.UserID, harvest.EmployeeID).Scan(&active)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrEmployeeNotFound
	}
	if err != nil {
		return fmt.Errorf("validar funcionário da colheita: %w", err)
	}
	if !active {
		return ErrEmployeeInactive
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO harvests (id, user_id, employee_id, harvest_date, created_by)
		VALUES ($1, $2, $3, $4, $2)
	`, harvest.ID, harvest.UserID, harvest.EmployeeID, harvest.HarvestDate)
	if err != nil {
		return fmt.Errorf("gravar cabeçalho da colheita: %w", err)
	}
	for _, item := range harvest.Items {
		_, err = tx.Exec(ctx, `
			INSERT INTO harvest_items (
				user_id, harvest_id, box_type, quantity,
				applied_rate_ciphertext, applied_rate_nonce, applied_rate_key_version
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, harvest.UserID, harvest.ID, string(item.BoxType), item.Quantity,
			item.AppliedAmount.Ciphertext, item.AppliedAmount.Nonce, item.AppliedAmount.KeyVersion,
		)
		if err != nil {
			return fmt.Errorf("gravar item da colheita: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("confirmar colheita: %w", err)
	}
	return nil
}

// ListHarvests retorna envelopes ainda cifrados; o Service só os abre depois de
// este SQL restringir a leitura ao dono autenticado e ao intervalo pedido.
func (r *Repository) ListHarvests(ctx context.Context, userID string, period HarvestPeriod) ([]ProtectedHarvestRecord, error) {
	const query = `
		SELECT h.id::text, h.employee_id::text, e.name, h.harvest_date,
		       hi.box_type, hi.quantity,
		       hi.applied_rate_ciphertext, hi.applied_rate_nonce,
		       hi.applied_rate_key_version
		FROM harvests h
		JOIN harvest_items hi
		  ON hi.user_id = h.user_id AND hi.harvest_id = h.id
		JOIN employees e
		  ON e.user_id = h.user_id AND e.id = h.employee_id
		WHERE h.user_id = $1
		  AND h.harvest_date >= $2::date
		  AND h.harvest_date <= $3::date
		ORDER BY h.harvest_date DESC, h.created_at DESC, h.id DESC, hi.box_type ASC
	`
	rows, err := r.pool.Query(ctx, query, userID, period.StartDate, period.EndDate)
	if err != nil {
		return nil, fmt.Errorf("listar lançamentos protegidos: %w", err)
	}
	defer rows.Close()

	harvests := make([]ProtectedHarvestRecord, 0)
	indices := make(map[string]int)
	for rows.Next() {
		var harvestID, employeeID, employeeName, boxType string
		var harvestDate time.Time
		var item ProtectedHarvestItem
		err := rows.Scan(
			&harvestID, &employeeID, &employeeName, &harvestDate,
			&boxType, &item.Quantity,
			&item.AppliedAmount.Ciphertext, &item.AppliedAmount.Nonce,
			&item.AppliedAmount.KeyVersion,
		)
		if err != nil {
			return nil, fmt.Errorf("ler item de lançamento protegido: %w", err)
		}

		index, exists := indices[harvestID]
		if !exists {
			index = len(harvests)
			indices[harvestID] = index
			harvests = append(harvests, ProtectedHarvestRecord{
				EmployeeName: employeeName,
				Harvest: ProtectedHarvest{
					ID: harvestID, UserID: userID, EmployeeID: employeeID,
					HarvestDate: calendarDate.Normalize(harvestDate),
					Items:       make([]ProtectedHarvestItem, 0, 2),
				},
			})
		}
		item.BoxType = BoxType(boxType)
		harvests[index].Harvest.Items = append(harvests[index].Harvest.Items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("listar lançamentos protegidos: %w", err)
	}
	return harvests, nil
}
