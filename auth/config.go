package auth

import (
	"regexp"
	"strings"
)

type Config struct {
	CacheSize           int `default:"1000"`
	Prefix              string
	URLValidReg         string
	ContentTypeValidReg string
	Headers             struct {
		AuthURL string `default:"X-Original-Uri"`
		IP      string `default:"X-Real-Ip"`
		Method  string `default:"X-Original-Method"`
	}
	DefaultAllow struct {
		ContentType string
		Method      string
		URL         string
	}
	ACL []ACLCfg
}

// For testCases
func (c *Config) SetDefauls() {
	c.CacheSize = 1000
	c.Headers.AuthURL = "X-Original-Uri"
	c.Headers.IP = "X-Real-Ip"
	c.Headers.Method = "X-Original-Method"
}

func (c *Config) getACLCompile() *[]ACLCompile {
	var acl_list []ACLCompile
	acl_list = make([]ACLCompile, len(c.ACL))
	for id, acl := range c.ACL {
		acl_comp := new(ACLCompile)
		acl_comp.Path = regexp.MustCompile(acl.Path)
		acl_comp.Users, acl_comp.AnyUsers = arrayToMap(acl.Users)
		acl_comp.Methods, acl_comp.AnyMethods = arrayToMap(acl.Methods)
		acl_comp.ContentType, acl_comp.AnyContentType = arrayToMap(acl.ContentType)

		acl_list[id] = *acl_comp
	}

	return &acl_list
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

/*
path: 000-0.sap-erp.db.operations.orders05.0
contenttype: application/vnd.kafka.binary.v2+json
users:
- name: sap
  allow: post


*/

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
