package funcionarios

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository concentra o acesso aos dados de funcionários usando o pool
// compartilhado da API, sem abrir um novo pool por funcionalidade.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository recebe um pool já aberto. Quem o criou continua responsável
// por fechá-lo, pois outras funcionalidades também podem utilizá-lo.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Create persiste a entidade montada e validada pelo Service sob um gerente existente.
// A validação de negócio fica no serviço; as restrições do banco continuam
// protegendo a integridade caso outro caminho tente gravar dados inválidos.
func (r *Repository) Create(ctx context.Context, employee Employee) (Employee, error) {
	// O ID é gerado pelo banco; os demais dados vêm da entidade do serviço.
	// RETURNING devolve a entidade persistida na mesma operação, sem SELECT extra.
	const query = `
		INSERT INTO employees (user_id, name, active)
		VALUES ($1, $2, $3)
		RETURNING id::text, user_id::text, name, active
	`

	// Parâmetros mantêm o nome separado do SQL. O contexto recebido preserva
	// o cancelamento e o prazo do chamador até a consulta ao PostgreSQL.
	err := r.pool.QueryRow(ctx, query, employee.UserID, employee.Name, employee.Active).Scan(
		&employee.ID,
		&employee.UserID,
		&employee.Name,
		&employee.Active,
	)
	if err != nil {
		return Employee{}, fmt.Errorf("criar funcionário: %w", err)
	}

	return employee, nil
}
