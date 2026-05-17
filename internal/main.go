package internal

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
)

// launchServer will execute a go routine to house the web server, it'll
// block until it's closed (from Main())
func launchServer(wg *sync.WaitGroup, server *http.Server) error {
	wg.Add(1)
	started, chErr := make(chan struct{}), make(chan error)
	go func() {
		defer wg.Done()
		close(started)
		if err := server.ListenAndServe(); err != nil {
			chErr <- err
		}
	}()
	<-started
	select {
	case <-time.After(10 * time.Second):
	case err := <-chErr:
		return err
	}
	return nil
}

// Main will execute the general business logic for the example
// all environmental components are provided as arguments
func Main(ctx context.Context, pwd string, args []string, envs map[string]string) error {
	var wg sync.WaitGroup

	fmt.Println("============================================")
	fmt.Printf("--go-blog-okta\n")
	fmt.Printf("--version:  %s\n", Version)
	fmt.Printf("--git branch: %s\n", GitBranch)
	fmt.Printf("--git commit:  %s\n", GitCommit)
	fmt.Println("============================================")

	//get configuration from environment
	config := NewConfiguration()
	config.Default()
	config.FromEnvs(envs)

	// create oidc provider
	provider, err := oidc.NewProvider(ctx, config.Issuer+"/")
	if err != nil {
		return err
	}

	//get public keys for token verification
	fmt.Printf("discovered JWKS URL: %s\n", provider.Endpoint().AuthURL) // Or use custom JWKS logic

	//create server/router and set handlers
	router := http.NewServeMux()
	server := &http.Server{
		Handler: router,
		Addr:    config.Address + ":" + config.Port,
	}
	router.HandleFunc("/", indexHandler(provider, config.Config))
	router.HandleFunc("/login", loginHandler(config))
	router.HandleFunc("/refreshing", refreshingHandler(config))
	router.HandleFunc("/verify", verifyHandler(provider, config.Config))
	router.HandleFunc("/authorization-code/callback", callbackHandler(provider, config))
	router.HandleFunc("/callback", callbackHandler(provider, config))
	router.HandleFunc("/api", apiHandler(ctx, config))
	router.HandleFunc("/logout", logoutHandler(config))

	//launch the server and wait for a ctrl+c
	if err := launchServer(&wg, server); err != nil {
		return err
	}
	<-ctx.Done()

	//close the server and wait for all the web server
	// go routine to return
	if err := server.Close(); err != nil {
		return err
	}
	wg.Wait()
	return nil
}
