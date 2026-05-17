package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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

func revokeRefreshToken(ctx context.Context, config *Configuration, refreshToken string) error {
	byts, err := json.Marshal(struct {
		ClientId     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		Token        string `json:"token"`
	}{
		ClientId:     config.ClientID,
		ClientSecret: config.ClientSecret,
		Token:        refreshToken,
	})
	if err != nil {
		return err
	}
	url := config.Issuer + "/oauth/revoke"
	req, err := http.NewRequestWithContext(ctx,
		http.MethodPost, url, bytes.NewBuffer(byts))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	switch response.StatusCode {
	default:
		return fmt.Errorf("unexepcted status code: %d", response.StatusCode)
	case http.StatusOK:
		return nil
	}
}
