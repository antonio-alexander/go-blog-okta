package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/antonio-alexander/go-blog-okta/internal"

	"github.com/joho/godotenv"
)

func init() {
	//this is janky, but it's here to simplify configuration so you
	// don't accidentally commit secrets
	if err := godotenv.Load("../.okta.env", ".env"); err != nil {
		fmt.Printf("error while godotenv.Load(): %s\n", err)
	}
}

func main() {
	pwd, _ := os.Getwd()
	args := os.Args[1:]
	envs := make(map[string]string)
	for _, env := range os.Environ() {
		if s := strings.Split(env, "="); len(s) > 1 {
			envs[s[0]] = strings.Join(s[1:], "=")
		}
	}
	chSignalInt := make(chan os.Signal, 1)
	signal.Notify(chSignalInt, os.Interrupt)
	if err := internal.Main(pwd, args, envs, chSignalInt); err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
	close(chSignalInt)
}
