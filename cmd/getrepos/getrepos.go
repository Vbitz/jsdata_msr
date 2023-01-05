package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/go-github/v48/github"
	"golang.org/x/oauth2"
)

var (
	accessTokenPath = flag.String("token", "tools/ghToken.txt", "The path to the GitHub access token.")
	outputFilename  = flag.String("output", "repos.njson", "The name of the output file to write to.")
	limit           = flag.Int("limit", 10000, "The number of repositories to request.")
)

func getAccessToken(path string) (string, error) {
	token, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(token), nil
}

func main() {
	flag.Parse()

	token, err := getAccessToken(*accessTokenPath)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	i := 0

	out, err := os.Create(*outputFilename)
	if err != nil {
		log.Fatal(err)
	}

	opts := &github.SearchOptions{
		Sort:  "stars",
		Order: "desc",
	}

outer:
	for {
		repos, resp, err := client.Search.Repositories(context.Background(), "language:typescript", opts)
		if err != nil {
			log.Printf("error executing search: %v", err)

			time.Sleep(10 * time.Second)

			continue
		}

		for _, repo := range repos.Repositories {
			repoObj := struct {
				Login      string
				Name       string
				GitUrl     string
				Stargazers int
			}{
				Login:      *repo.Owner.Login,
				Name:       *repo.Name,
				GitUrl:     *repo.GitURL,
				Stargazers: *repo.StargazersCount,
			}

			log.Printf("[rate: %d/%d] %d/%d %+v", resp.Rate.Remaining, resp.Rate.Limit, i, *limit, repoObj)

			bytes, err := json.Marshal(repoObj)
			if err != nil {
				log.Fatal(err)
			}

			fmt.Fprintf(out, "%s\n", string(bytes))

			i++

			if i > *limit {
				break outer
			}
		}

		opts.Page = resp.NextPage

		time.Sleep(5 * time.Second)
	}
}
