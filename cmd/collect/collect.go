package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/filesystem"
)

var (
	repoList   = flag.String("repos", "repos.njson", "A newline delimited JSON file containing a list of repositories to download.")
	outputFile = flag.String("outputFile", "output.csv", "The newline delimitated JSON file to write to.")
)

var counter = make(map[string]uint64)

func count(id string) {
	counter[id] += 1
}

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
	if err != nil {
		return nil, err
	}

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

type PackageJson struct {
	Name            string `json:"name"`
	Author          interface{}
	Version         string `json:"version"`
	Repository      interface{}
	DevDependencies map[string]string `json:"devDependencies"`
	Dependencies    map[string]string `json:"dependencies"`
}

func parsePackageJson(contents string) (PackageJson, error) {
	var ret PackageJson

	err := json.Unmarshal([]byte(contents), &ret)
	if err != nil {
		return PackageJson{}, err
	}

	return ret, nil
}

func getTsVersion(packageJson PackageJson) string {
	var tsVersion string

	if ver, ok := packageJson.Dependencies["typescript"]; ok {
		tsVersion = ver
	} else {
		if ver, ok := packageJson.DevDependencies["typescript"]; ok {
			tsVersion = ver
		}
	}

	tsVersion = strings.TrimPrefix(tsVersion, " ")
	tsVersion = strings.TrimPrefix(tsVersion, "~")
	tsVersion = strings.TrimPrefix(tsVersion, "^")

	return tsVersion
}

type CommitData struct {
	Id                string
	Date              uint64
	Hash              string
	TypeScriptVersion string
}

func collectDataCommit(id string, commit *object.Commit) (CommitData, error) {
	tree, err := commit.Tree()
	if err != nil {
		return CommitData{}, err
	}

	packageJsonFile, err := tree.File("package.json")
	if err != nil {
		return CommitData{}, err
	}

	pkgContents, err := packageJsonFile.Contents()
	if err != nil {
		return CommitData{}, err
	}

	packageJson, err := parsePackageJson(pkgContents)
	if err != nil {
		return CommitData{}, err
	}

	tsVersion := getTsVersion(packageJson)

	return CommitData{
		Id:                id,
		Date:              uint64(commit.Author.When.Unix()),
		Hash:              commit.Hash.String(),
		TypeScriptVersion: tsVersion,
	}, nil
}

type RepoData struct {
	Id      string
	Commits []CommitData
}

func collectData(id string, repo *git.Repository) (RepoData, error) {
	var ret RepoData
	commitIter, err := repo.CommitObjects()
	if err != nil {
		return RepoData{}, err
	}
	for {
		commit, err := commitIter.Next()

		if err == io.EOF {
			break
		} else if err != nil {
			return RepoData{}, err
		}

		count("commit")

		commitData, err := collectDataCommit(id, commit)
		if err != nil {
			count("collectError")
			continue
		}

		ret.Commits = append(ret.Commits, commitData)
	}
	return ret, nil
}

type RepoLine struct {
	Login      string
	Name       string
	GitUrl     string
	Stargazers int
}

func main() {
	flag.Parse()

	repoList, err := os.Open(*repoList)
	if err != nil {
		log.Fatal(err)
	}

	scan := bufio.NewScanner(repoList)

	output, err := os.Create(*outputFile)
	if err != nil {
		log.Fatal(err)
	}

	csvOutput := csv.NewWriter(output)

	visited := make(map[string]bool)

	rows := 0

	for scan.Scan() {
		var line RepoLine

		err := json.Unmarshal(scan.Bytes(), &line)
		if err != nil {
			log.Printf("error unmarshaling: %v", err)
			continue
		}

		id := fmt.Sprintf("%s/%s", line.Login, line.Name)

		if _, ok := visited[id]; ok {
			continue
		}

		visited[id] = true

		log.Printf("[%d] Repo %s/%s", rows, line.Login, line.Name)

		repo, err := cloneRepo(line.GitUrl)
		if err != nil {
			log.Printf("error opening: %v", err)
			continue
		}

		count("opened")

		commitData, err := collectData(id, repo)
		if err != nil {
			count("failure")
			log.Printf("error collecting data: %v", err)
			continue
		}

		for _, commit := range commitData.Commits {
			rows += 1
			err := csvOutput.Write([]string{
				commit.Id, commit.TypeScriptVersion, fmt.Sprintf("%d", commit.Date),
			})
			if err != nil {
				log.Fatal(err)
			}
		}

		count("success")
	}

	for k, v := range counter {
		log.Printf("[C] %s = %d", k, v)
	}
}
