package auth

import (
	"fmt"
	"regexp"
)

// Структура запроса проверки
type AuthRequest struct {
	AuthURL     string
	AuthUser    string
	IP          string
	Method      string
	ContentType string
}

type authCheck func(c *AuthRequest) error

type SimpleAccessController struct {
	checks []authCheck
}

func (a *SimpleAccessController) addCheck(ac authCheck) {
	if ac != nil {
		a.checks = append(a.checks, ac)
	}
}

func NewSimpleAccessController(c *Config) (AccessController, error) {
	ac := new(SimpleAccessController)

	ac.makeChecks(c)

	return ac, nil
}

func (a *SimpleAccessController) makeChecks(cfg *Config) {
	a.checks = make([]authCheck, 0)
	a.addCheck(a.getURLValidReg(cfg))
	// Last
	// a.addCheck(a.getContentTypeValidReg(cfg))
	acl := cfg.getAclRulesCompiled()
	a.addCheck(a.getACLVerifier(acl))
}

func (a *SimpleAccessController) Validate(authRequest *AuthRequest) error {
	for _, authCheckFn := range a.checks {
		if err := authCheckFn(authRequest); err != nil {
			return err
		}
	}
	return nil
}

func (a *SimpleAccessController) getURLValidReg(cfg *Config) authCheck {
	if len(cfg.URLValidReg) > 0 {
		urlRe := regexp.MustCompile(cfg.URLValidReg)

		return func(authRequest *AuthRequest) error {
			if !urlRe.MatchString(authRequest.AuthURL) {
				return fmt.Errorf("Incorrect URL")
			}
			return nil
		}
	}
	return nil
}

func (a *SimpleAccessController) getContentTypeValidReg(cfg *Config) authCheck {
	if len(cfg.ContentTypeValidReg) > 0 {
		ctRe := regexp.MustCompile(cfg.ContentTypeValidReg)
		return func(authRequest *AuthRequest) error {
			if !ctRe.MatchString(authRequest.ContentType) {
				return fmt.Errorf("Incorrect Content-Type")
			}
			return nil
		}
	}
	return nil
}

func (a *SimpleAccessController) getACLVerifier(acl []*aclRuleCompilded) authCheck {
	if len(acl) > 0 {
		return func(authRequest *AuthRequest) error {
			var err error

			for cnt := range acl {
				aclEl := acl[len(acl)-1-cnt]
				if aclEl.IsMatch(authRequest.AuthURL, authRequest.AuthUser, authRequest.Method) {
					err = aclEl.IsAllow(authRequest.ContentType)
					if err != nil {
						return fmt.Errorf("%w [%d]", err, len(acl)-1-cnt)
					}
					return nil // allow
				}
			}

			return fmt.Errorf("Permission denied")
		}
	}
	return nil
}
