package auth

import (
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hashicorp/vault/api"
)

type VaultConfig struct {
	Addr          string `default:"http://vault:8200"`
	Token         string
	Path          string
	Refresh       int    `default:"5"`
	ServiceHeader string `default:"RA-Service"`

	SkipUsers bool `default:"true"`
}

type vaultAuth struct {
	config   *VaultConfig
	client   *api.Client
	services map[string]map[string]string
}

func CreateVaultAuth(c *VaultConfig) (error, *vaultAuth) {
	if len(c.Token) < 1 {
		return errors.New("No VaultToken"), nil
	}
	if len(c.Path) < 1 {
		return errors.New("No Vault Path"), nil
	}
	va := new(vaultAuth)
	va.config = c
	if err := va.setClient(); err != nil {
		return err, nil
	}

	if err := va.loadSecrets(); err != nil {
		return err, nil
	}
	log.Println("Config Addr ")

	return nil, va
}

func (va *vaultAuth) GetMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		username, ok := va.Auth(c.GetHeader(va.config.ServiceHeader), strings.TrimPrefix(c.GetHeader("Authorization"), "Basic "))
		log.Println("AUTH:", ok, "User:", username)
	}
}

func (va *vaultAuth) getService(service_header string) map[string]string {
	if service_header == "" && len(va.services) == 1 {
		for key := range va.services {
			return va.services[key]
		}
	}
	if service, ok := va.services[service_header]; ok {
		return service
	}
	log.Println("Service not found: ", service_header)
	return nil
}

func (va *vaultAuth) Auth(service string, token string) (string, bool) {
	if _, ok := va.services[service]; ok {
		if user, ok := va.services[service][token]; ok {
			return user, true
		}
	}
	return "", false
}

func (va *vaultAuth) setClient() error {
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		return err
	}
	client.SetAddress(va.config.Addr)
	client.SetToken(va.config.Token)
	va.client = client
	return nil
}

func (va *vaultAuth) loadSecrets() error {
	var services_list []string
	var err error
	services := make(map[string]map[string]string)
	services_list, err = va.listPath(va.config.Path)
	if err != nil {
		return err
	}
	for _, service := range services_list {
		// Recurse if path is a directory
		if strings.HasSuffix(service, "/") {
			service_name := strings.TrimSuffix(service, "/")
			log.Println("Service: ", service_name)
			services[service_name], err = va.loadSecretsForService(fmt.Sprint(va.config.Path, service))
			if err != nil {
				return err
			}
		}
	}

	va.services = services
	return nil
}

func (va *vaultAuth) loadSecretsForService(path string) (map[string]string, error) {
	var secretStore map[string]string
	var users_list []string
	var data interface{}
	var err error
	secretStore = make(map[string]string)
	users_list, err = va.listPath(path)
	if err != nil {
		return nil, err
	}
	for _, user := range users_list {
		if !strings.HasSuffix(user, "/") {
			data, err = va.readValueWithKey(fmt.Sprint(path, user), "password")
			if err != nil {
				if va.config.SkipUsers {
					log.Println("Error load user(skiping):", err.Error())
				} else {
					return nil, err
				}
			}
			secretStore[createBase64Token(user, data.(string))] = user
			log.Println("Register user:", user, "token:", createBase64Token(user, data.(string)))
		}
	}

	return secretStore, nil
}

func createBase64Token(user string, password string) string {
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", user, password)))
}

func (va *vaultAuth) listPath(path string) ([]string, error) {
	var items []string
	var err error

	secret, err := va.client.Logical().List(path)
	if err != nil {
		return nil, err
	}

	if secret == nil {
		return nil, nil
	}

	for _, path := range secret.Data {
		// expecting "[secret0 secret1 secret2...]"
		//
		// if the name both exists as directory and as file
		// e.g. "/secret/" and "/secret" it will print an empty line
		items = strings.Split(strings.Trim(fmt.Sprint(path), "[]"), " ")
	}
	return items, nil
}

func (va *vaultAuth) readValueWithKey(path string, key string) (interface{}, error) {
	data, err := va.readValues(path)
	if err != nil {
		return nil, err
	}

	if _, ok := data[key]; ok {
		return data[key], nil
	}
	return nil, errors.New("Key: " + key + " - does not exists")
}

func (va *vaultAuth) readValues(path string) (map[string]interface{}, error) {
	secret, err := va.client.Logical().Read(path)
	if err != nil {
		return nil, err
	}
	if secret == nil {
		return nil, errors.New("Secret does not exist")
	}

	return secret.Data, nil
}

func (va *vaultAuth) listKV() ([]string, error) {
	mounts, err := va.client.Sys().ListMounts()
	if err != nil {
		return nil, err
	}

	var backends []string

	for x, i := range mounts {
		// With the 0.8.3 release of vault the "generic" backend was renamed to "kv". For backwards
		// compatibility consider both of them
		if i.Type == "kv" || i.Type == "generic" {
			backends = append(backends, x)
		}
	}
	return backends, nil
}

// /<namespace>/username:password
/* Vault policy:
 * path "service_name/ra/" {
 *   capabilities = [ "read", "list" ]
 * }
 *
 * path "service_name/ra/+/+" {
 *   capabilities = [ "read", "list" ]
 * }
 */
