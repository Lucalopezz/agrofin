package auth

// Manager contém somente os dados prontos para persistência; a senha original
// não atravessa a fronteira entre Service e Repository.
type Manager struct {
	LoginName    string
	PasswordHash string
}
