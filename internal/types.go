package internal

const (
	cookieKeyAccessToken       string = "access_token"
	cookieKeyOAuthState        string = "oauth_state"
	cookieKeyOAuthCodeVerifier string = "oauth_code_verifier"
)

// These variables are populated at build time
// to find where the variables are...use  go tool nm ./app | grep app
// REFERENCE: https://www.digitalocean.com/community/tutorials/using-ldflags-to-set-version-information-for-go-applications
var (
	Version   string
	GitCommit string
	GitBranch string
)
