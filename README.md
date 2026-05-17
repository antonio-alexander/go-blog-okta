# go-blog-okta (auth0) (github.com/antonio-alexander/go-blog-okta)

The purpose of this repository is to attempt to distill what I've learned about Okta/Auth0 and how to create an incredibly basic implementation. It's based on the [okta-gin-sample](https://github.com/okta-samples/okta-go-gin-sample); sans the gin...a virgin okta-go-sample if you will. I think [gin web framework](https://github.com/gin-gonic/gin) is incredible, it simplifies a lot, for better or for worse.

I think some of the samples fail in that they sacrifice simplicity (and readability) for a REALLY good presentation and high functionality with gin. I think from the perspective of the output, gin is a clear winner, but from the perspective of understanding what the code is doing and figuring out how to integrate it into your (probably) existing, well architected code; it fails. If you're not using gin or you have static assets; the sample fails you in that you have to separate the functional code from; hopefully the combination of this document and _my_ sample code will save you the effort.

## Bibliography

- [https://developer.okta.com/docs/guides/sign-into-web-app-redirect/go/main/](https://developer.okta.com/docs/guides/sign-into-web-app-redirect/go/main/)
- [https://gist.github.com/logrusorgru/abd846adb521a6fb39c7405f32fec0cf](https://gist.github.com/logrusorgru/abd846adb521a6fb39c7405f32fec0cf)
- [https://stackoverflow.com/questions/57172415/how-to-send-cookie-through-redirection-in-golang](https://stackoverflow.com/questions/57172415/how-to-send-cookie-through-redirection-in-golang)
- [https://community.auth0.com/t/receiving-encrypted-jwts-jwe-instead-of-rs256-signed-jwts-with-regular-web-app-and-oauth/181744](https://community.auth0.com/t/receiving-encrypted-jwts-jwe-instead-of-rs256-signed-jwts-with-regular-web-app-and-oauth/181744)
- [https://github.com/MicahParks/keyfunc](https://github.com/MicahParks/keyfunc)

## Getting Started

The Okta documentation is incredible; please use it; if you get to the point where you can create an .okta.env you'll be able to start what's below:

```sh
make build
make okta-envs
make run
```

> Your _gitignored_ `.okta.env` should contain the following values:  OKTA_OAUTH2_ISSUER, OKTA_OAUTH2_CLIENT_ID, OKTA_OAUTH2_CLIENT_SECRET, OKTA_OAUTH2_AUDIENCE. You should be able to grab all of these values from your okta/auth0 dashboard

After you build and run the example image, you can use a browser to navigate to [http://localhost:8080](http://localhost:8080) to have the web server validate if there's an access token saved as a cookie. From there you can navigate to [http://localhost:8080/login](http://localhost:8080/login) where it will perform a 302 redirect to Okta to login which will then redirect you back to the local webserver to perform the remainder of the Okta process and generate the access token.

The [http://localhost:8080/logout](http://localhost:8080/logout) can be used to delete the access token from the stored cookie and redo the Okta process.

## Okta/Auth0 Process

The process of requesting an identity token via Okta is really straight forward, if you're familiar with OAuth2 as it's just about identical (there are a couple more fields that you may not have used). In general, you'll follow the authorization PKCE (Proof Key for Code Exchange) flow which is as follows:

1. generate a unique oauth state (used to validate you're processing the right callback/redirect)
2. generate an oauth code verifier and challenge (for PKCE)
3. use the oauth configuration to perform a 302 redirect to login to okta (it will append the challenge and method)
4. receive code via redirect/callback after a successful okta login
5. use okta configuration to perform a code exchange using the code verifier and get three tokens: a raw idt token, a refresh token and an access token (both JWTs)
6. verify the raw idt token

This is ONLY the functional portion of the okta effort, it doesn't include how data is shared via the request; this is done using [cookies](https://go.dev/src/net/http/cookie.go) or [sessions](https://github.com/gorilla/sessions) which is **deprecated**.

> There are other ways to store the cookie, but it should be considered privileged and [secure] cookies are the least complicated solution to make them available to any application within your domain

## Application Process

The application process is a bit complicated. I think the examples (especially integrating with gin) obfuscate what you need to do at a minimum to support this in your application. In short what your business logic is trying to accomplish is:

1. confirm if you have a locally cached access token (that's valid) and if so, return that access token as-is
2. if you don't have a valid token, but have a valid refresh token, use the refreshing flow to generate a _new_ access token
3. if you don't have a valid refresh token, or that token is expired, perform the authorization code flow to get a new access and refresh token

In general, I think this can be accomplished by having a handful of http handlers:

- /

> this is your _index.html_ and it can be used to communicate whether you have a valid cached (cookie) access or refresh token

- /login

> this performs the part of the authorization code flow to get the authorization code, it will generate some data to start and then redirect to okta (see [okta process step 3](#oktaauth0-process)). In addition, this handler should be intelligent enough to know NOT to redirect to okta if the token is valid

- /callback

> this takes the authorization code and performs a token exchange (see [okta process step 5](#oktaauth0-process)), validates the token and then caches it in a cookie

- /logout

> this just deletes the cached tokens and revokes the refresh token which logs the user out, any attempts to re-use the site by that browser will require the full authorization flow via /login

- /refreshing

> this explicitly performs the refreshing flow (for testing purposes)

- /api

> this simulates how an API would validate that an access token is valid

## Decoding/Validating Tokens

Truly _secure_ tokens, are signed using asymmetric keys, meaning there's a public and a private key. The private key is using to sign the token and the public key can be used to validate that the token was signed using that specific private key. In more complex systems, this public key is rotated such that different keys must be verified with different public keys.

> It's not necessary to verify a token with the public key in order to read it, the JWT is not encrypted **ONLY** signed

OIDC has a well-known jwks endpoint (e.g., `<ISSUER>/.well-known/jwks.json`). that can be used to grab the public key with a known contract. This contract contains an array of keys, each with a specific id (a key id if you will). When an access token is generated, it's claims contains a field called kid that can be used to determine which key to use to verify a given token.

If there's an API that has to verify identities, they'll do so for every token that's received. Because the public keys are rotated (including **new** public keys); you'll need to cache them in-memory and periodically refresh that cache. The [github.com/MicahParks/keyfunc](github.com/MicahParks/keyfunc) package includes logic to do this and a callback function that can be used to inject your own cache.

## Gotchas/Things to Keep in Mind

Although I was able to put together this example pretty quickly and pull it apart, here is a list of things that I did wrong the first time:

- I had the wrong redirect uri configured

> This is something that can sneak by you, okta will enforce that you have a valid redirect uri/url, if you don't the callback part of the process will fail, be sure to go into your web application configuration and set appropriate values for the redirect.

- Using okta's JWT verifier is a fool’s errand

> See: [https://github.com/okta/okta-jwt-verifier-golang/pull/123](https://github.com/okta/okta-jwt-verifier-golang/pull/123) Okta/Auth0 is very particular about the trailing slash in the issuer, but within this code it ALSO uses the issuer to construct the URLs creating situations where the URL involves multiple slashes like `http://something.com//authorize`

- I was initially receiving JWEs instead of JWTs

> See: [https://community.auth0.com/t/receiving-encrypted-jwts-jwe-instead-of-rs256-signed-jwts-with-regular-web-app-and-oauth/181744](https://community.auth0.com/t/receiving-encrypted-jwts-jwe-instead-of-rs256-signed-jwts-with-regular-web-app-and-oauth/181744) The short version is that you need to include an audience with your initial authorization call so you get a JWT instead of a JWE

- It didn't make a lot of sense that there were so many tokens

> This is very confusing at first, but each token has a different intended audience and will obfuscate or omit data accordingly; access tokens are expected to be consumed by a specific API while the raw idt tokens are expected to be consumed by the application that initiated the flow (and would probably perform the revocation)

## Frequently Asked Questions

- If someone has a valid refresh token, can they use it to get a valid access token from anywhere?

> Yes, even though they might not have the client id/secret to talk to okta/authn directly, you most likely will have an externally facing API that's accessible by anyone so there's no real way to avoid it

- What happens when an access token expires?

> When an access token expires, most _sane_ libraries will simply fail when you attempt to validate them, stating that the token has expired (even though it can still be validated using the public key). Keep in mind that access tokens are **stateless** and in general, can't be revoked

- What happens when a refresh token expires or is revoked while still in use?

> When a refresh token expires or is revoked, it can no longer be used to complete the refreshing code flow. The refresh token is opaque in that it contains no information inside of it, but it's stored server-side (i.e., Okta/Auth0) and it knows if a refresh token is still valid and who it belongs to

- How do I revoke access tokens?

> For Okta/Auth0, access tokens can't be revoked, it's a little impractical to revoke access tokens, so in general, we simply have a short expiration of those access tokens. BUT, it's totally possible to revoke access tokens if you have somewhere you can store those tokens until they expire. You would forward that access token to an application server-side that can perform that check

- Are refresh tokens dangerous to store client side?

> Yes and no, if a refresh token is successfully exfiltrated it can be used until its revoked and it may be slightly difficult to determine if it's being used in two places (but this has a relatively obvious signature). Rotating secrets can help mitigate this and make the signature mentioned earlier _more_ obvious

- Is a refresh token that hasn't expired or been revoked still functional?

> Yes, just because it hasn't been used recently doesn't reduce its function.  it's a good idea to revoke refresh tokens that haven't been used recently to force non-active users to sign in again

- Why am I not getting a refresh token?

> Refresh tokens have to be enabled explicitly by creating an API/application within Okta/Auth0

- Why do I get three tokens (raw idt, refresh token and access token)?

> Each token has a different purpose and an intended audience, the raw idt is meant to be consumed by the client/browser and the application performing the authorization code/refreshing flow, while the refresh token is meant to be used to generate new access tokens and the access token itself is meant to be consumed by an application validating the identity. This is generally communicated in the audience field of the token, but otherwise can be confusing since they can all be validated by the same public key

- Why should I create access tokens with short expirations?

> Short expirations limit the usefulness of the stateless tokens. The shorter the expiration, the less exposure if that token is leaked. If you decide to revoke these stateless tokens, you have to store those tokens until they expire. IF you have a longer expiration time you have to store those tokens for the duration

- Should I have an expiration on my refresh tokens?

> Yes, although indefinite is convenient, its complicated. I think rotation of refresh tokens and automatic revocation of idle tokens are the best solution to mitigate the downside of expirations etc.

- Client-side, how would I handle a situation where the access token expires or is about to expire? How can I avoid using it and getting an error I could avoid having to feel confident parsing?

> This is much easier with more complex languages like Typescript, but you could probably write similar code in JavaScript for a simple website in an application. But in general, you would extract the expiration date from the claims and give the user an option to refresh the token when it’s about to expire which would then perform the refreshing flow to update the token. You could also do it automatically when a user navigates away from a certain page (you could do the refreshing flow in the background). OR...finally, you could wait until there's a validation error from an api endpoint that needs the token and force the refreshing flow

- Should you use refresh token rotation?

> Yes, refresh token rotation mitigates situations where refresh tokens are exfiltrated and can be used (indefinitely) to generate access tokens. When you rotate the refresh token, it will automatically revoke the previous resource token while giving you a new one with a new access token. Unless the entity with the exfiltrated token constantly performs the refreshing flow, eventually they'll be left with only a valid access token (with a short expiration) and no valid refresh token. As they still lack the user id/password, they can't perform the authorization code flow to get a _new_ refresh token from scratch, they have to exfiltrate it again
