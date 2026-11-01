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

// List consulta os funcionários vinculados ao gerente informado.
func (r *Repository) List(ctx context.Context, userID string) ([]Employee, error) {
	const query = `
		SELECT id::text, user_id::text, name, active
		FROM employees
		WHERE user_id = $1
		ORDER BY name, id
	`

	// O filtro limita a consulta ao dono dos dados. O ID desempata nomes iguais,
	// mantendo a ordem previsível sem confundir pessoas com o mesmo nome.
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("listar funcionários: %w", err)
	}
	defer rows.Close()
	// Uma lista sem registros deve ser representada como [], não null.
	employees := make([]Employee, 0)

	for rows.Next() {
		var employee Employee
		err := rows.Scan(
			&employee.ID,
			&employee.UserID,
			&employee.Name,
			&employee.Active,
		)
		if err != nil {
			return nil, fmt.Errorf("ler funcionário: %w", err)
		}

		employees = append(employees, employee)
	}

	// Next também retorna false quando a leitura falha. Distinguir esse caso
	// do término normal impede devolver uma lista incompleta como sucesso.
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterar funcionários: %w", err)
	}

	return employees, nil
}
