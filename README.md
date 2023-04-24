# go-blog-okta (github.com/antonio-alexander/go-blog-kafka)

The purpose of this repository is to attempt to distill what I've learned about Okta and how to create an incredibly basic implementation. It's based on the [okta-gin-sample](https://github.com/okta-samples/okta-go-gin-sample); sans the gin...a virgin okta-go-sample if you will. I think [gin web framework](https://github.com/gin-gonic/gin) is incredible, it simplifies a lot, for better or for worse.

I think some of the samples fail in that they sacrifice simplicity (and readibility) for a REALLY good presentation and high functionality with gin. I think from the perspective of the output, gin is a clear winner, but from the perspective of understanding what the code is doing and figuring out how to integrate it into your (probably) integrated code; it fails. If you're not using gin or you have static assets; the sample fails you in that you have to separate the functional code from; hopefully the combination of this document and _my_ sample code will save you the effort.

## Getting Started

The Okta documentation is incredible; please use it; if you get to the point where you can create an .okta.env you'll be able to start what's below:

```sh
make build
make okta-envs
make run
```

After you build and run the example image, you can use a browser to navigate to [http://localhost:8080](http://localhost:8080) to have the web server validate if there's an access token saved as a cookie. From there you can navigate to [http://localhost:8080/login](http://localhost:8080/login) where it will perform a 302 redirect to Okta to login which will then redirect you back to the local webserver to perform the remainder of the Okta process and generate the access token.

The [http://localhost:8080/logout](http://localhost:8080/logout) can be used to delete the access token from the stored cookie and redo the Okta process.

## Okta Process

The process of requesting an identity token via Okta is really straight forward, if you're familiar with OAuth2 as it's just about identical (there are a couple more fields that you may not have used). In general, you'll follow the authorization PKCE (Proof Key for Code Exchange) flow which is as follows:

1. generate a unique oauth state (used to validate you're processing the right callback/redirect)
2. generate an oauth code verifier and challenge (for PKCE)
3. use the oauth configuration to perform a 302 redirect to login to okta (it will append the challenge and method)
4. receive code via redirect/callback after a successful okta login
5. use okta configuration to perform a code exchange using the code verifier and get a token
6. verify the token

This is ONLY the functional portion of the okta effort, it doesn't include how data is shared via the request; this is done using [cookies](https://go.dev/src/net/http/cookie.go) or [sessions](https://github.com/gorilla/sessions) which is **deprecated**.

## Application Process

The application process is a bit complicated. I think the examples (especially integrating with gin) obfuscate what you need to do at a minium to support this in your application. In short what your business logic is trying to accomplish is:

1. confirm if you've have a locally cached token (that's valid) and if so, return that identity
2. if you don't have a locally cached token (that's valid); redirect to oauth and attempt to login
3. receive the authorization code from okta via the redirect uri

In general, I think this can be accomplished by having a handful of http handlers:

- /

> this is your _index.html_ and it can be used to communicate whether you have a valid cached (cookie) token or not

- /login

> this performs the part of the authorization code flow to get the authorization code, it will generate some data to start and then redirect to okta (see [okta process step 3](#okta-process)). In addition, this handler should be intelligent enough to know NOT to redirect to okta if the token is valid

- /callback

> this takes the authorization code and performs a token exchange (see [okta process step 5](#okta-process)), validates the token and then caches it in a cookie

- /logout

> this just deletes the cached token to effectively _log the user out_

## Gotchas/Things to Keep in Mind

Although I was able to put together this example pretty quickly and pull it apart, here is a list of things that I did wrong the first time:

- I had the wrong redirect uri configured

> This is something that can sneak by you, okta will enforce that you have a valid redirect uri/url, if you don't the callback part of the process will fail, be sure to go into your web application configuration and set appropriate values for the redirect.

## Bibliography

- [https://developer.okta.com/docs/guides/sign-into-web-app-redirect/go/main/](https://developer.okta.com/docs/guides/sign-into-web-app-redirect/go/main/)
- [https://gist.github.com/logrusorgru/abd846adb521a6fb39c7405f32fec0cf](https://gist.github.com/logrusorgru/abd846adb521a6fb39c7405f32fec0cf)
- [https://stackoverflow.com/questions/57172415/how-to-send-cookie-through-redirection-in-golang](https://stackoverflow.com/questions/57172415/how-to-send-cookie-through-redirection-in-golang)
