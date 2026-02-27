package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"os"

	awslambda "github.com/aws/aws-lambda-go/lambda"
	"github.com/stahnma/gh-flox/internal/commands"
	"github.com/stahnma/gh-flox/internal/config"
	lambdapkg "github.com/stahnma/gh-flox/internal/lambda"
)

//go:embed additional_repos.json
var additionalReposJSON []byte

var (
	GitSHA   string
	GitDirty string
)

func main() {
	cfg := config.FromEnvironment()

	var additionalRepos []string
	if err := json.Unmarshal(additionalReposJSON, &additionalRepos); err != nil {
		log.Fatalf("Error parsing additional repositories JSON: %v", err)
	}

	app, err := commands.NewApp(cfg, additionalRepos, GitSHA, GitDirty)
	if err != nil {
		log.Fatalf("Error initializing application: %v", err)
	}

	if os.Getenv("LAMBDA_TASK_ROOT") != "" {
		awslambda.Start(lambdapkg.NewHandler(app))
	} else {
		rootCmd := app.NewRootCommand()
		if err := rootCmd.Execute(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		if err := app.SaveCache(); err != nil {
			log.Fatalf("Error saving cache: %v", err)
		}
	}
}
