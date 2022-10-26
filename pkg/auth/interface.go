package auth

type AccessController interface {
	Validate(authRequest *AuthRequest) error
}
