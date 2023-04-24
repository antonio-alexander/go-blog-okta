package internal

import (
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
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
func Main(pwd string, args []string, envs map[string]string, chSignalInt chan os.Signal) error {
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

	//create server/router and set handlers
	router := http.NewServeMux()
	server := &http.Server{
		Handler: router,
		Addr:    config.Address + ":" + config.Port,
	}
	router.HandleFunc("/", indexHandler())
	router.HandleFunc("/login", loginHandler(config))
	router.HandleFunc("/authorization-code/callback", callbackHandler(config))
	router.HandleFunc("/callback", callbackHandler(config))
	router.HandleFunc("/logout", logoutHandler())

	//launch the server and wait for a ctrl+c
	if err := launchServer(&wg, server); err != nil {
		return err
	}
	<-chSignalInt

	//close the server and wait for all the web server
	// go routine to return
	if err := server.Close(); err != nil {
		return err
	}
	wg.Wait()
	return nil
}
