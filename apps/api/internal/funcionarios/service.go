package funcionarios

import "context"

// A interface descreve somente a persistência usada pelo serviço. Ela permite
// testar as regras sem PostgreSQL, mantendo o SQL no Repository concreto.
type employeeCreator interface {
	Create(ctx context.Context, employee Employee) (Employee, error)
}

// Service coordena o cadastro e aplica suas regras antes da persistência.
type Service struct {
	repo employeeCreator
}

// NewService recebe a persistência sem abrir conexões nem guardar um contexto:
// cada chamada de Create tem seu próprio prazo e cancelamento.
func NewService(repo employeeCreator) *Service {
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
