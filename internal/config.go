package internal

import "golang.org/x/oauth2"

const (
	defaultRedirectUri string = "http://localhost:8080/authorization-code/callback"
	defaultAddress     string = ""
	defaultPort        string = "8080"
)

// Configuration is a combination of the configuration needed to execute
// the example code; it contains configuratio nfor the web server and the
// Okta/OAuth configuration
type Configuration struct {
	oauth2.Config
	Address     string
	Port        string
	Issuer      string
	RedirectUri string
	Audience    string
}

// NewConfiguration can be used to create a new instance
// of Configuration with the internal pointers created
func NewConfiguration() *Configuration {
	c := &Configuration{}
	c.Default()
	return c
}

// Default will populate the internal pointers with default
// values
func (c *Configuration) Default() {
	if c == nil {
		return
	}
	c.RedirectURL = defaultRedirectUri
	c.Address = defaultAddress
	c.Port = defaultPort
	c.Scopes = []string{"openid", "profile", "email", "offline_access"}
	c.Endpoint.AuthStyle = oauth2.AuthStyleInParams
	c.RedirectURL = defaultRedirectUri
}

// FromEnvs will populate a set number of values within
// configuration from environmental variables
func (c *Configuration) FromEnvs(envs map[string]string) {
	if c == nil {
		return
	}
	if s, ok := envs["OKTA_OAUTH2_REDIRECT_URI"]; ok {
		c.RedirectURL = s
	}
	if s, ok := envs["OKTA_OAUTH2_CLIENT_ID"]; ok {
		c.ClientID = s
	}
	if s, ok := envs["OKTA_OAUTH2_CLIENT_SECRET"]; ok {
		c.ClientSecret = s
	}
	if s, ok := envs["OKTA_OAUTH2_AUDIENCE"]; ok {
		c.Audience = s
	}
	if s, ok := envs["OKTA_OAUTH2_ISSUER"]; ok {
		c.Issuer = s
		c.Endpoint.AuthURL = s + "/authorize"
		c.Endpoint.TokenURL = s + "/oauth/token"
	}
}
