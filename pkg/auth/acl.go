package auth

import (
	"fmt"
	"regexp"
	"strings"
)

// Правило
type AclRule struct {
	Path        string
	Users       []string
	Methods     []string
	ContentType []string
}
type aclRuleCompilded struct {
	Path           *regexp.Regexp
	Users          map[string]bool
	Methods        map[string]bool
	ContentType    map[string]bool
	AnyUsers       bool
	AnyMethods     bool
	AnyContentType bool
}

// Функция создает новый список правил на основе списка ACL из конфигурации
func (c *Config) getAclRulesCompiled() []*aclRuleCompilded {
	aclList := make([]*aclRuleCompilded, len(c.ACL))
	for id, acl := range c.ACL {
		aRC := new(aclRuleCompilded)
		aRC.Path = regexp.MustCompile(acl.Path)
		aRC.Users, aRC.AnyUsers = arrayToMap(acl.Users)
		aRC.Methods, aRC.AnyMethods = arrayToMap(acl.Methods)
		aRC.ContentType, aRC.AnyContentType = arrayToMap(acl.ContentType)

		aclList[id] = aRC
	}

	return aclList
}

func arrayToMap(in []string) (rez map[string]bool, any bool) {
	any = false
	rez = make(map[string]bool, len(in))

	for _, key := range in {
		rez[strings.ToLower(key)] = true
		if strings.EqualFold(key, "any") {
			any = true
		}
	}
	return
}

func (ac *aclRuleCompilded) IsUrlMatch(url string) bool {
	return ac.Path.MatchString(url)
}

func (ac *aclRuleCompilded) IsAllow(content_type, username, method string) error {
	if _, ok := ac.Users[username]; !ok && !ac.AnyUsers {
		return fmt.Errorf("Username not allowed")
	}
	if _, ok := ac.Methods[strings.ToLower(method)]; !ok && !ac.AnyMethods {
		return fmt.Errorf("Method not allowed")
	}
	if _, ok := ac.ContentType[strings.ToLower(content_type)]; !ok && !ac.AnyContentType {
		return fmt.Errorf("Content-Type not allowed")
	}
	return nil
}
