package funcionarios

import "context"

// A interface descreve as operações de persistência usadas pelo serviço.
// Ela permite testar o serviço sem depender do PostgreSQL.
type employeeRepository interface {
	Create(ctx context.Context, employee Employee) (Employee, error)
	List(ctx context.Context, userID string) ([]Employee, error)
	Rename(ctx context.Context, employee Employee) (Employee, error)
	Deactivate(ctx context.Context, employee Employee) error
}

// Service coordena os casos de uso do cadastro de funcionários.
type Service struct {
	repo employeeRepository
}

// NewService recebe a persistência sem abrir conexões nem guardar um contexto:
// cada operação recebe o prazo e o cancelamento do seu chamador.
func NewService(repo employeeRepository) *Service {
	return &Service{repo: repo}
}

// Create valida o nome e monta a entidade antes de solicitar sua gravação.
// userID é fornecido pelo chamador confiável, separado do nome do cadastro.
func (s *Service) Create(ctx context.Context, userID, name string) (Employee, error) {
	// A identidade vem de um chamador confiável (o handler após middleware); o
	// nome, por vir do cliente, precisa passar pelas regras do domínio.
	name, err := normalizeName(name)
	if err != nil {
		return Employee{}, err
	}

	// O serviço define o estado inicial do domínio. O gerente vem do chamador
	// confiável; o ID fica vazio até o banco gerar o identificador na inserção.
	employee := Employee{
		UserID: userID,
		Name:   name,
		Active: true,
	}

	// A persistência recebe a entidade pronta e o contexto da operação original.
	return s.repo.Create(ctx, employee)
}

// List consulta os funcionários do gerente identificado pelo chamador confiável.
func (s *Service) List(ctx context.Context, userID string) ([]Employee, error) {
	return s.repo.List(ctx, userID)
}

// Rename valida o novo nome antes da persistência; userID vem do chamador
// confiável e employeeID identifica o registro que ele deseja alterar.
func (s *Service) Rename(ctx context.Context, userID, employeeID, name string) (Employee, error) {
	name, err := normalizeName(name)
	if err != nil {
		return Employee{}, err
	}

	// A operação usa somente identificação e nome. O UPDATE não grava Active:
	// o estado atual é preservado e devolvido pelo RETURNING do banco.
	employee := Employee{ID: employeeID, UserID: userID, Name: name}
	return s.repo.Rename(ctx, employee)
}

// Deactivate preserva o cadastro e seu histórico, tornando-o inativo.
// userID vem do chamador confiável, identificado pela sessão.
func (s *Service) Deactivate(ctx context.Context, userID, employeeID string) error {
	// O serviço escolhe o estado de destino; não recebe um active do cliente.
	// O nome não participa desta operação e não deve ser sobrescrito.
	employee := Employee{ID: employeeID, UserID: userID, Active: false}
	return s.repo.Deactivate(ctx, employee)
}
