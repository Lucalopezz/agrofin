package auth

import "context"

type managerCreator interface {
	CreateManager(ctx context.Context, manager Manager) error
}

// Service aplica as regras de cadastro do gerente antes de persistir a entidade.
type Service struct {
	repo managerCreator
}

// NewService recebe a persistência sem abrir conexões nem guardar um contexto;
// cada cadastro propaga o contexto da própria requisição.
func NewService(repo managerCreator) *Service {
	return &Service{repo: repo}
}

// RegisterManager normaliza e valida as credenciais, gera o hash e só então
// entrega ao Repository a entidade pronta para gravação.
func (s *Service) RegisterManager(ctx context.Context, loginName, password string) error {
	loginName, err := normalizeLoginName(loginName)
	if err != nil {
		return err
	}

	passwordHash, err := createPasswordHash(password, loginName)
	if err != nil {
		return err
	}

	manager := Manager{
		LoginName:    loginName,
		PasswordHash: passwordHash,
	}
	return s.repo.CreateManager(ctx, manager)
}
