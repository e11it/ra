package auth

import (
	"fmt"
	"regexp"
	"strings"
)

type ACLCfg struct {
	Path        string
	Users       []string
	Methods     []string
	ContentType []string
}

type ACLCompile struct {
	Path           *regexp.Regexp
	Users          map[string]bool
	Methods        map[string]bool
	ContentType    map[string]bool
	AnyUsers       bool
	AnyMethods     bool
	AnyContentType bool
}

func (ac *ACLCompile) IsMatch(url string) bool {
	return ac.Path.MatchString(url)
}

func (ac *ACLCompile) IsAllow(user string, method string, content_type string) error {
	if _, ok := ac.Users[user]; !ok && !ac.AnyUsers {
		return fmt.Errorf("User not allowed")
	}

	if _, ok := ac.Methods[strings.ToLower(method)]; !ok && !ac.AnyMethods {
		return fmt.Errorf("Method not allowed")
	}

	if _, ok := ac.ContentType[strings.ToLower(content_type)]; !ok && !ac.AnyContentType {
		return fmt.Errorf("Content-Type not allowed")
	}
	return nil
}
