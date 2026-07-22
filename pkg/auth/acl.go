package auth

import (
	"fmt"
	"regexp"
	"strings"
)

// Правило
type ACLRule struct {
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
func (c *Config) getAclRulesCompiled() ([]*aclRuleCompilded, error) {
	aclList := make([]*aclRuleCompilded, len(c.ACL))
	for id, acl := range c.ACL {
		aRC := new(aclRuleCompilded)
		path, err := regexp.Compile(acl.Path)
		if err != nil {
			return nil, fmt.Errorf("compile acl[%d] path regexp: %w", id, err)
		}
		aRC.Path = path
		aRC.Users, aRC.AnyUsers = arrayToMap(acl.Users)
		aRC.Methods, aRC.AnyMethods = arrayToMapLower(acl.Methods)
		aRC.ContentType, aRC.AnyContentType = arrayToMapLower(acl.ContentType)

		aclList[id] = aRC
	}

	return aclList, nil
}

func arrayToMap(in []string) (rez map[string]bool, hasAny bool) {
	hasAny = false
	rez = make(map[string]bool, len(in))

	for _, key := range in {
		rez[key] = true
		if strings.EqualFold(key, "any") {
			hasAny = true
		}
	}
	return
}

func arrayToMapLower(in []string) (rez map[string]bool, hasAny bool) {
	hasAny = false
	rez = make(map[string]bool, len(in))

	for _, key := range in {
		rez[strings.ToLower(key)] = true
		if strings.EqualFold(key, "any") {
			hasAny = true
		}
	}
	return
}

func (ac *aclRuleCompilded) IsURLMatch(url string) bool {
	return ac.Path.MatchString(url)
}

func (ac *aclRuleCompilded) IsAllow(content_type, username, method string) error {
	if _, ok := ac.Users[username]; !ok && !ac.AnyUsers {
		return fmt.Errorf("username not allowed")
	}
	if _, ok := ac.Methods[strings.ToLower(method)]; !ok && !ac.AnyMethods {
		return fmt.Errorf("method not allowed")
	}
	if _, ok := ac.ContentType[strings.ToLower(content_type)]; !ok && !ac.AnyContentType {
		return fmt.Errorf("content-type not allowed")
	}
	return nil
}
