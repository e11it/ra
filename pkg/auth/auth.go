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
	if c == nil {
		return nil, fmt.Errorf("auth config is nil")
	}
	if len(c.ACL) == 0 {
		return nil, fmt.Errorf("auth acl must contain at least one rule")
	}

	ac := new(SimpleAccessController)
	if err := ac.makeChecks(c); err != nil {
		return nil, err
	}

	return ac, nil
}

func (a *SimpleAccessController) makeChecks(cfg *Config) error {
	a.checks = make([]authCheck, 0)
	urlCheck, err := a.getURLValidReg(cfg)
	if err != nil {
		return err
	}
	a.addCheck(urlCheck)
	acl, err := cfg.getAclRulesCompiled()
	if err != nil {
		return err
	}
	a.addCheck(a.getACLVerifier(acl))
	return nil
}

func (a *SimpleAccessController) Validate(authRequest *AuthRequest) error {
	for _, authCheckFn := range a.checks {
		if err := authCheckFn(authRequest); err != nil {
			return err
		}
	}
	return nil
}

func (a *SimpleAccessController) getURLValidReg(cfg *Config) (authCheck, error) {
	if len(cfg.URLValidReg) > 0 {
		urlRe, err := regexp.Compile(cfg.URLValidReg)
		if err != nil {
			return nil, fmt.Errorf("compile url validation regexp: %w", err)
		}

		return func(authRequest *AuthRequest) error {
			if !urlRe.MatchString(authRequest.AuthURL) {
				return fmt.Errorf("incorrect url")
			}
			return nil
		}, nil
	}
	return nil, nil
}

func (a *SimpleAccessController) getACLVerifier(acl []*aclRuleCompilded) authCheck {
	return func(authRequest *AuthRequest) error {
		var err, lastError error

		lastError = fmt.Errorf("permission denied: no allow acl for this topic")
		for cnt, aclEl := range acl {
			if aclEl.IsURLMatch(authRequest.AuthURL) {
				err = aclEl.IsAllow(
					authRequest.ContentType,
					authRequest.AuthUser,
					authRequest.Method)
				if err == nil {
					return nil // allow
				}
				// save error and check next acl rules for success
				lastError = fmt.Errorf("%w [%d]", err, cnt)
			}
		}

		return lastError
	}
}
