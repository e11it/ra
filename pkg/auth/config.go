package auth

// Консигурация
type Config struct {
	Prefix              string
	URLValidReg         string
	ContentTypeValidReg string
	DefaultAllow        struct {
		ContentType string
		Method      string
		URL         string
	}
	ACL []ACLRule
}
