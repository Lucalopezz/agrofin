package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"time"
)

type managerCreator interface {
	CreateManager(ctx context.Context, manager Manager) error
}

// authRepository reúne somente as operações de persistência que o serviço de
// autenticação precisa. O Service decide as regras; o Repository executa SQL.
type authRepository interface {
	managerCreator
	FindManager(ctx context.Context, loginName string) (Manager, error)
	FindManagerByID(ctx context.Context, userID string) (Manager, error)
	FindManagerProfile(ctx context.Context, userID string) (ManagerProfile, error)
	UpdateManagerName(ctx context.Context, manager Manager) (ManagerProfile, error)
	UpdateManagerPassword(ctx context.Context, change ManagerPasswordChange) error
	SetManagerActive(ctx context.Context, manager Manager) error
	CreateSession(ctx context.Context, session Session) error
	ResolveSession(ctx context.Context, tokenHash []byte) (Principal, error)
	RevokeSession(ctx context.Context, tokenHash []byte) error
}

const (
	// Argon2id consome memória de propósito. O limite evita que várias requisições
	// simultâneas multipliquem esse custo até esgotar a memória do processo.
	maximumConcurrentPasswordOperations = 2
	// Este é o prazo absoluto. O prazo por inatividade é verificado pelo Repository.
	sessionLifetime = 30 * 24 * time.Hour
)

// Service aplica as regras de cadastro do gerente antes de persistir a entidade.
type Service struct {
	repo         authRepository
	passwordWork chan struct{}
}

// NewService recebe a persistência sem abrir conexões nem guardar um contexto;
// cada cadastro propaga o contexto da própria requisição.
func NewService(repo authRepository) *Service {
	return &Service{
		repo:         repo,
		passwordWork: make(chan struct{}, maximumConcurrentPasswordOperations),
	}
}

// RegisterManager normaliza e valida as credenciais, gera o hash e só então
// entrega ao Repository a entidade pronta para gravação.
func (s *Service) RegisterManager(ctx context.Context, name, loginName, password string) error {
	// Nome público e login têm regras distintas: o nome pode conter espaços e
	// acentos; o login continua normalizado pelo alfabeto definido em password.go.
	name, err := normalizeManagerName(name)
	if err != nil {
		return err
	}
	// Normalizar antes de comparar ou persistir garante que o mesmo login não tenha
	// variantes por espaços ou diferenças entre maiúsculas e minúsculas.
	loginName, err = normalizeLoginName(loginName)
	if err != nil {
		return err
	}
	if err := s.acquirePasswordWork(ctx); err != nil {
		return err
	}
	defer s.releasePasswordWork()

	// A senha original é usada apenas para produzir o hash; a entidade que atravessa
	// a fronteira do Repository contém o hash, nunca a senha em texto claro.
	passwordHash, err := createPasswordHash(password, loginName)
	if err != nil {
		return err
	}

	manager := Manager{
		Name:         name,
		LoginName:    loginName,
		PasswordHash: passwordHash,
		// Cada cliente recebe uma conta ativa; os dados serão vinculados ao seu ID.
		Active: true,
	}
	return s.repo.CreateManager(ctx, manager)
}

// GetManagerProfile busca os dados públicos do gerente autenticado, sem ler hash de senha.
func (s *Service) GetManagerProfile(ctx context.Context, userID string) (ManagerProfile, error) {
	return s.repo.FindManagerProfile(ctx, userID)
}

// UpdateManagerName valida o novo nome e entrega ao Repository uma entidade pronta.
// LoginName fica deliberadamente fora da entidade parcial para não ser alterado.
func (s *Service) UpdateManagerName(ctx context.Context, userID, name string) (ManagerProfile, error) {
	name, err := normalizeManagerName(name)
	if err != nil {
		return ManagerProfile{}, err
	}
	return s.repo.UpdateManagerName(ctx, Manager{ID: userID, Name: name})
}

// ChangePassword confirma a senha atual, valida e calcula o hash da nova senha
// antes de pedir a persistência. Após salvar, o Repository revoga todas as sessões.
func (s *Service) ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	// Verificar a senha atual e calcular a nova envolve Argon2id; a mesma barreira
	// do login mantém esse trabalho limitado a duas operações simultâneas.
	if err := s.acquirePasswordWork(ctx); err != nil {
		return err
	}
	defer s.releasePasswordWork()

	// Procurar o gerente pelo ID da sessão evita que um usuário malicioso tente trocar a
	// senha de outra conta apenas conhecendo o login_name. O Repository compara o hash
	manager, err := s.repo.FindManagerByID(ctx, userID)
	if errors.Is(err, errManagerNotFound) {
		return ErrInvalidSession
	}
	if err != nil {
		return err
	}
	if !manager.Active {
		return ErrInvalidSession
	}

	currentPassword, err = normalizePasswordInput(currentPassword)
	if err != nil || !verifyPasswordHash(manager.PasswordHash, currentPassword) {
		// O principal já identifica a conta; ainda assim, uma senha atual incorreta
		// não deve revelar detalhes do hash ou do estado interno da conta.
		return ErrInvalidCredentials
	}

	newPassword, err = validateNewPassword(newPassword, manager.LoginName)
	if err != nil {
		return err
	}
	if newPassword == currentPassword {
		return invalidInput("a nova senha deve ser diferente da senha atual")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	newPasswordHash, err := hashPasswordMaterial(newPassword)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	// O Repository compara o hash anterior dentro da transação. Isso impede que
	// duas trocas concorrentes sobrescrevam silenciosamente uma à outra.
	return s.repo.UpdateManagerPassword(ctx, ManagerPasswordChange{
		UserID:               manager.ID,
		PreviousPasswordHash: manager.PasswordHash,
		NewPasswordHash:      newPasswordHash,
	})
}

// SetManagerActive aceita desativação por uma sessão válida; a reativação ainda
// exige operador, pois a própria conta não consegue autenticar desativada.
func (s *Service) SetManagerActive(ctx context.Context, userID string, active bool) error {
	if active {
		return ErrActivationRequiresOperator
	}
	return s.repo.SetManagerActive(ctx, Manager{ID: userID, Active: false})
}

// Login verifica as credenciais sem revelar se o login existe e cria uma sessão opaca.
func (s *Service) Login(ctx context.Context, loginName, password string) (createdSession, error) {
	// A vaga é adquirida antes de qualquer trabalho de autenticação porque tanto
	// verificar uma senha quanto calcular o hash fictício consomem memória e CPU.
	if err := s.acquirePasswordWork(ctx); err != nil {
		return createdSession{}, err
	}
	defer s.releasePasswordWork()

	// Credenciais malformadas e usuário inexistente seguem um caminho com custo
	// Argon2id semelhante ao de uma senha incorreta, reduzindo enumeração por tempo.
	loginName, loginErr := normalizeLoginName(loginName)
	password, passwordErr := normalizePasswordInput(password)
	if loginErr != nil || passwordErr != nil {
		performDummyPasswordHash(password)
		return createdSession{}, ErrInvalidCredentials
	}

	manager, err := s.repo.FindManager(ctx, loginName)
	if errors.Is(err, errManagerNotFound) {
		performDummyPasswordHash(password)
		return createdSession{}, ErrInvalidCredentials
	}
	if err != nil {
		return createdSession{}, err
	}
	// Não informamos se a senha estava certa para uma conta desativada: a resposta
	// externa permanece igual para usuário inexistente, senha incorreta ou inativo.
	if !verifyPasswordHash(manager.PasswordHash, password) || !manager.Active {
		return createdSession{}, ErrInvalidCredentials
	}
	if err := ctx.Err(); err != nil {
		return createdSession{}, err
	}
	// O navegador recebe o token original uma única vez. O banco recebe apenas seu
	// SHA-256, então uma cópia isolada do banco não contém um cookie reutilizável.
	tokenBytes := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, tokenBytes); err != nil {
		return createdSession{}, errors.New("não foi possível iniciar a sessão")
	}
	token := base64.RawURLEncoding.EncodeToString(tokenBytes)
	tokenHash := sha256.Sum256(tokenBytes)
	expiresAt := time.Now().UTC().Add(sessionLifetime)
	session := Session{
		UserID:    manager.ID,
		TokenHash: tokenHash[:],
		ExpiresAt: expiresAt,
	}
	if err := s.repo.CreateSession(ctx, session); err != nil {
		return createdSession{}, err
	}
	return createdSession{Token: token, ExpiresAt: expiresAt}, nil
}

// AuthenticateSession valida o token opaco e recupera o principal exclusivamente do banco.
func (s *Service) AuthenticateSession(ctx context.Context, token string) (Principal, error) {
	// A forma e o tamanho são conferidos antes do banco para rejeitar cookies
	// aleatórios/malformados sem consulta e sem tratar o token como identidade.
	tokenBytes, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(tokenBytes) != 32 {
		return Principal{}, ErrInvalidSession
	}
	tokenHash := sha256.Sum256(tokenBytes)
	return s.repo.ResolveSession(ctx, tokenHash[:])
}

// Logout revoga a sessão no servidor; um cookie ausente ou malformado já está efetivamente encerrado.
func (s *Service) Logout(ctx context.Context, token string) error {
	// Logout é idempotente: cookie ausente ou inválido significa que não há uma
	// sessão utilizável para revogar, mas o Handler ainda vai expirar o cookie.
	tokenBytes, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(tokenBytes) != 32 {
		return nil
	}
	tokenHash := sha256.Sum256(tokenBytes)
	return s.repo.RevokeSession(ctx, tokenHash[:])
}

func (s *Service) acquirePasswordWork(ctx context.Context) error {
	// Não guardamos ctx no Service: cada chamada usa o contexto da requisição que
	// pode ser cancelada quando o cliente desconecta ou o servidor atinge o prazo.
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case s.passwordWork <- struct{}{}:
		if err := ctx.Err(); err != nil {
			s.releasePasswordWork()
			return err
		}
		return nil
	default:
		// Preferimos falhar rapidamente a enfileirar trabalho de alto custo sem limite.
		return ErrAuthenticationBusy
	}
}

func (s *Service) releasePasswordWork() {
	<-s.passwordWork
}
