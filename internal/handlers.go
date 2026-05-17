package internal

import (
	"context"
	"fmt"
	"net/http"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/golang-jwt/jwt/v5"
	"github.com/okta/okta-jwt-verifier-golang/utils"
	"github.com/thanhpk/randstr"
	"golang.org/x/oauth2"
)

// indexHandler will execute a handler that will have different output
// depending on whether or not the access token is stored within the
// value cookie
func indexHandler(provider *oidc.Provider, config oauth2.Config) func(w http.ResponseWriter, r *http.Request) {
	verifier := provider.Verifier(&oidc.Config{
		ClientID: config.ClientID,
	})
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("  Executing index (/) handler")
		rawIdToken := readCookie(r, cookieKeyRawIdtoken)
		if _, err := verifier.Verify(r.Context(), rawIdToken); err != nil {
			//redirect to login
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		_, _ = w.Write(fmt.Appendf(nil, "valid raw id token: %s", rawIdToken))
	}
}

func logoutHandler(config *Configuration) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("  Executing logout (/logout) handler")

		//revoke token
		refreshToken := readCookie(r, cookieKeyRefreshToken)
		if refreshToken != "" {
			if err := revokeRefreshToken(r.Context(),
				config, refreshToken); err != nil {
				handleError(w, err)
				return
			}
		}
		//unset the access token cookie
		deleteCookie(w, cookieKeyAccessToken)

		//unset the refresh token cookie
		deleteCookie(w, cookieKeyRefreshToken)

		//unset the raw idt token cookie
		deleteCookie(w, cookieKeyRawIdtoken)

		//redirect back to index
		http.Redirect(w, r, "/", http.StatusFound)
	}
}

func loginHandler(config *Configuration) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("  Executing login (/login) handler")

		//REFERENCE: https://github.com/okta/samples-golang/issues/20
		r.Header.Add("Cache-Control", "no-cache")

		refreshToken := readCookie(r, cookieKeyRefreshToken)
		if refreshToken != "" {
			//execute refreshing flow
			if token, err := config.TokenSource(r.Context(),
				&oauth2.Token{RefreshToken: refreshToken}).Token(); err != nil {
				fmt.Printf("unable to use refreshing flow: %s", err.Error())
				//continue on, but delete the invalid refresh token
				deleteCookie(w, cookieKeyRefreshToken)
			} else {
				//write the access token to the client via cookie
				writeCookie(w, cookieKeyAccessToken, token.AccessToken)

				//redirect to /verify
				http.Redirect(w, r, "/verify", http.StatusFound)
			}
		}

		// Generate a random state parameter for CSRF security
		oauthState := randstr.Hex(16)
		fmt.Printf("state generated: %s\n", oauthState)

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
			oauth2.SetAuthURLParam("audience", config.Audience),
		), http.StatusFound)
	}
}

func callbackHandler(provider *oidc.Provider, config *Configuration) func(w http.ResponseWriter, r *http.Request) {
	verifier := provider.Verifier(&oidc.Config{
		ClientID: config.ClientID,
	})
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
			r.Context(),
			r.URL.Query().Get("code"),
			oauth2.SetAuthURLParam("code_verifier", readCookie(r, cookieKeyOAuthCodeVerifier)),
		)
		if err != nil {
			handleError(w, err, http.StatusUnauthorized)
			return
		}

		// Extract the ID Token from OAuth2 token.
		rawIdToken, ok := token.Extra("id_token").(string)
		if !ok {
			handleError(w, fmt.Errorf("id token missing from OAuth2 token"), http.StatusUnauthorized)
			return
		}

		//verify the token
		if _, err := verifier.Verify(r.Context(), rawIdToken); err != nil {
			handleError(w, err, http.StatusForbidden)
			return
		}

		//write the raw id to the client via cookie
		writeCookie(w, cookieKeyRawIdtoken, rawIdToken)

		//write the access token to the client via cookie
		writeCookie(w, cookieKeyAccessToken, token.AccessToken)

		//write the refresh token to the client via cookie
		if token.RefreshToken != "" {
			writeCookie(w, cookieKeyRefreshToken, token.RefreshToken)
		}

		//redirect back to verify
		http.Redirect(w, r, "/verify", http.StatusFound)
	}
}

func verifyHandler(provider *oidc.Provider, config oauth2.Config) func(w http.ResponseWriter, r *http.Request) {
	verifier := provider.Verifier(&oidc.Config{
		ClientID: config.ClientID,
	})
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("  Executing verifier (/verifier) handler")

		//read the raw id token
		rawIdToken := readCookie(r, cookieKeyRawIdtoken)

		//verify the token
		if _, err := verifier.Verify(r.Context(), rawIdToken); err != nil {
			handleError(w, err, http.StatusForbidden)
			return
		}

		//provide positive feedback
		_, _ = w.Write(fmt.Appendf(nil, "valid raw id token: %s", rawIdToken))
	}
}

func apiHandler(ctx context.Context, config *Configuration) func(w http.ResponseWriter, r *http.Request) {
	k, err := keyfunc.NewDefaultCtx(ctx, []string{config.Issuer + "/.well-known/jwks.json"})
	if err != nil {
		panic(err)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("  Executing api (/api) handler")

		//get access token from cookies
		accessToken := readCookie(r, cookieKeyAccessToken)

		// 2. Parse and verify the token
		token, err := jwt.Parse(accessToken, k.Keyfunc)
		if err != nil {
			handleError(w, err, http.StatusForbidden)
			return
		}

		if token.Valid {
			_, _ = w.Write(fmt.Appendf(nil, "valid access token: %s", accessToken))
			// Access claims: token.Claims.(jwt.MapClaims)
		} else {
			_, _ = w.Write(fmt.Append(nil, "invalid access token"))
		}
	}
}

func refreshingHandler(config *Configuration) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("  Executing login (/refreshing) handler")

		//REFERENCE: https://github.com/okta/samples-golang/issues/20
		r.Header.Add("Cache-Control", "no-cache")

		refreshToken := readCookie(r, cookieKeyRefreshToken)
		if refreshToken == "" {
			_, _ = w.Write(fmt.Append(nil, "invalid refresh token"))
			return
		}
		//execute refreshing flow
		token, err := config.TokenSource(r.Context(),
			&oauth2.Token{RefreshToken: refreshToken}).Token()
		if err != nil {
			handleError(w, err, http.StatusForbidden)
			return
		}

		//write the refresh token (in case it's being rotated/replaced)
		writeCookie(w, cookieKeyRefreshToken, token.RefreshToken)

		//write the access token to the client via cookie
		writeCookie(w, cookieKeyAccessToken, token.AccessToken)

		//redirect to /verify
		http.Redirect(w, r, "/verify", http.StatusFound)
	}
}
