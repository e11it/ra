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
	ACL []AclRule
}

/*
path: 000-0.sap-erp.db.operations.orders05.0
contenttype: application/vnd.kafka.binary.v2+json
users:
- name: sap
  allow: post


*/

/* TODO: REMOVE. Looks like unused
type parsedConfig struct {
	Prefix       string
	DefaultAllow struct {
		ContentType *regexp.Regexp
		Method      *regexp.Regexp
		URL         *regexp.Regexp
	}
}

func (c *Config) getParsedConfig() (*parsedConfig, error) {
	var err error
	pc := new(parsedConfig)
	pc.Prefix = c.Prefix
	if pc.DefaultAllow.ContentType, err = regexp.Compile(c.DefaultAllow.ContentType); err != nil {
		return nil, err
	}
	if pc.DefaultAllow.Method, err = regexp.Compile(c.DefaultAllow.Method); err != nil {
		return nil, err
	}
	if pc.DefaultAllow.URL, err = regexp.Compile(c.DefaultAllow.URL); err != nil {
		return nil, err
	}
	return pc, nil
}
*/
