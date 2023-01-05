package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"log"
	"net/url"
	"os"
	"path"
	"runtime/pprof"
	"strings"

	"github.com/go-git/go-billy/v5/osfs"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/storage/filesystem"
)

var (
	repoList   = flag.String("repos", "repos.njson", "A newline delimited JSON file containing a list of repositories to download.")
	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
)

func repoUrlToStore(repoUrl string) (string, error) {
	parsed, err := url.Parse(repoUrl)
	if err != nil {
		return "", err
	}
	storePath := strings.TrimPrefix(parsed.Path, "/")
	storePath = strings.TrimSuffix(storePath, ".git")
	return path.Join("store", storePath), nil
}

func cloneRepo(repoUrl string) (*git.Repository, error) {
	storePath, err := repoUrlToStore(repoUrl)
	if err != nil {
		return nil, err
	}

	store := filesystem.NewStorage(osfs.New(storePath), cache.NewObjectLRU(8*1024*1024))

	if _, err := os.Stat(storePath); err == nil {
		repo, err := git.Open(store, nil)
		if err != nil {
			return nil, err
		}
		return repo, nil
	} else {
		repo, err := git.Clone(store, nil, &git.CloneOptions{
			URL:      strings.Replace(repoUrl, "git://", "https://", 1),
			Progress: os.Stdout,
		})
		if err != nil {
			return nil, err
		}
		return repo, nil
	}
}

type RepoLine struct {
	Login      string
	Name       string
	GitUrl     string
	Stargazers int
}

func main() {
	flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	repoList, err := os.Open(*repoList)
	if err != nil {
		log.Fatal(err)
	}

	scan := bufio.NewScanner(repoList)

	for scan.Scan() {
		var line RepoLine

		err := json.Unmarshal(scan.Bytes(), &line)
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("Downloading: %+v", line)

		_, err = cloneRepo(line.GitUrl)
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("repo = %s/%s", line.Login, line.Name)
	}

	// commits, err := repo.CommitObjects()
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// for {
	// 	commit, err := commits.Next()
	// 	if err == io.EOF {
	// 		break
	// 	} else if err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	tree, err := commit.Tree()
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	files := tree.Files()
	// 	totalFiles := 0
	// 	for {
	// 		_, err := files.Next()
	// 		if err == io.EOF {
	// 			break
	// 		} else if err != nil {
	// 			log.Fatal(err)
	// 		}
	// 		totalFiles += 1
	// 	}
	// 	log.Printf("%s %d", commit.Hash, totalFiles)
	// }
}
