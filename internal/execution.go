package internal

import (
	"fmt"
	"net/http"

	verifier "github.com/okta/okta-jwt-verifier-golang"
)

// readCookie can be used to read the value for a given cookie
func readCookie(r *http.Request, key string) string {
	cookie, err := r.Cookie(key)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// writeCookie can be used to write a given key/value to a cookie as long
// as set the maxAge value; by default it'll be 0
func writeCookie(w http.ResponseWriter, key, value string, maxAges ...int) {
	maxAge := int(0)
	if len(maxAges) > 0 {
		maxAge = maxAges[0]
	}
	http.SetCookie(w, &http.Cookie{
		Name:   key,
		Value:  value,
		MaxAge: maxAge,
		Path:   "/",
	})
}

// deleteCookie can be used to remove a cookie by "unsetting"
// it's value and setting the maxAge to -1
func deleteCookie(w http.ResponseWriter, key string) {
	http.SetCookie(w, &http.Cookie{
		Name:   key,
		Value:  "",
		MaxAge: -1,
	})
}

// verifyToken can be used to verify a given access token and output
// a verifier and error
func verifyToken(config *Configuration, accessToken string) (*verifier.Jwt, error) {
	jv := verifier.JwtVerifier{
		Issuer: config.Issuer,
		ClaimsToValidate: map[string]string{
			"aud": config.ClientID,
		},
	}
	result, err := jv.New().VerifyIdToken(accessToken)
	switch {
	case err != nil:
		return nil, err
	case result == nil:
		return nil, fmt.Errorf("token could not be verified")
	}
	return result, nil
}

// handleError can be used to output an error and status code
// when an error occurs
func handleError(w http.ResponseWriter, err error, statusCodes ...int) {
	statusCode := http.StatusInternalServerError
	if len(statusCodes) > 0 {
		statusCode = statusCodes[0]
	}
	w.WriteHeader(statusCode)
	_, _ = w.Write([]byte(err.Error()))
}
