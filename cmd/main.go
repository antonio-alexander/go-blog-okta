package main

import (
	"context"
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
	if err := godotenv.Load(".okta.env"); err != nil {
		fmt.Printf("error while godotenv.Load(): %s\n", err)
	}
}

func main() {
	pwd, _ := os.Getwd()
	args := os.Args[1:]
	envs := make(map[string]string)
	for _, env := range os.Environ() {
		if key, value, ok := strings.Cut(env, "="); ok && value != "" {
			envs[key] = value
		}
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	if err := internal.Main(ctx, pwd, args, envs); err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
}
