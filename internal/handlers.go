package internal

import (
	"context"
	"fmt"
	"net/http"

	"github.com/okta/okta-jwt-verifier-golang/utils"
	"github.com/thanhpk/randstr"
	"golang.org/x/oauth2"
)

// indexHandler will execute a handler that will have different output
// depending on whether or not the access token is stored within the
// value cookie
func indexHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("  Executing index (/) handler")
		switch accessToken := readCookie(r, cookieKeyAccessToken); accessToken {
		default:
			_, _ = w.Write([]byte(fmt.Sprintf("access token found: %s", accessToken)))
		case "":
			_, _ = w.Write([]byte("access token not found"))
		}
	}
}

func logoutHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("  Executing logout (/logout) handler")

		//unset the access token cookie
		deleteCookie(w, cookieKeyAccessToken)

		//redirect back to index
		http.Redirect(w, r, "/", http.StatusFound)
	}
}

func loginHandler(config *Configuration) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("  Executing login (/login) handler")

		//REFERENCE: https://github.com/okta/samples-golang/issues/20
		r.Header.Add("Cache-Control", "no-cache")

		// Generate a random state parameter for CSRF security
		oauthState := randstr.Hex(16)

		// Create the PKCE code verifier and code challenge
		oauthCodeVerifier, err := utils.GenerateCodeVerifierWithLength(50)
		if err != nil {
			handleError(w, err)
			return
		}

		// get sha256 hash of the code verifier
		oauthCodeChallenge := oauthCodeVerifier.CodeChallengeS256()

		//set the oauth2 state
		writeCookie(w, cookieKeyOAuthState, oauthState, 60)

		// set the oauth code verifier
		writeCookie(w, cookieKeyOAuthCodeVerifier, oauthCodeVerifier.String(), 60)

		//redirect to Okta login PKCE flow
		http.Redirect(w, r, config.AuthCodeURL(
			oauthState,
			oauth2.SetAuthURLParam("code_challenge", oauthCodeChallenge),
			oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		), http.StatusFound)
	}
}

// callbackHandler
func callbackHandler(config *Configuration) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("  Executing callback (/callback) handler")

		// Check the state that was returned in the query string is the same as the above state
		if r.URL.Query().Get("state") == "" || r.URL.Query().Get("state") != readCookie(r, cookieKeyOAuthState) {
			handleError(w, fmt.Errorf("the state was not as expected"), http.StatusForbidden)
			return
		}

		// Make sure the code was provided
		if r.URL.Query().Get("error") != "" {
			handleError(w, fmt.Errorf("authorization server returned an error: %s", r.URL.Query().Get("error")), http.StatusForbidden)
			return
		}

		// Make sure the code was provided
		if r.URL.Query().Get("code") == "" {
			handleError(w, fmt.Errorf("the code was not returned or is not accessible"), http.StatusForbidden)
			return
		}

		token, err := config.Exchange(
			context.Background(),
			r.URL.Query().Get("code"),
			oauth2.SetAuthURLParam("code_verifier", readCookie(r, cookieKeyOAuthCodeVerifier)),
		)
		if err != nil {
			handleError(w, err, http.StatusUnauthorized)
			return
		}

		// Extract the ID Token from OAuth2 token.
		rawIDToken, ok := token.Extra("id_token").(string)
		if !ok {
			handleError(w, fmt.Errorf("id token missing from OAuth2 token"), http.StatusUnauthorized)
			return
		}

		//verify the token
		_, err = verifyToken(config, rawIDToken)
		if err != nil {
			handleError(w, err, http.StatusForbidden)
			return
		}

		//write the access token to the client via cookie
		writeCookie(w, cookieKeyAccessToken, token.AccessToken)

		//redirect back to index
		http.Redirect(w, r, "/", http.StatusFound)
	}
}
