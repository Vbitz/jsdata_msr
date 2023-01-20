package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"

	"example.com/jsdata/v3/pkg/common"
)

var (
	input  = flag.String("input", "repos.njson", "The input list of repos.")
	limit  = flag.Int("limit", 500, "The number of repos to extract.")
	output = flag.String("output", "repos.clean.njson", "The cleaned list of repos to output.")
)

type RepoList []common.RepoLine

// Len implements sort.Interface
func (l *RepoList) Len() int {
	return len(*l)
}

// Less implements sort.Interface
func (l *RepoList) Less(i int, j int) bool {
	return (*l)[i].Stargazers > (*l)[j].Stargazers
}

// Swap implements sort.Interface
func (l *RepoList) Swap(i int, j int) {
	(*l)[i], (*l)[j] = (*l)[j], (*l)[i]
}

func main() {
	flag.Parse()

	input, err := os.Open(*input)
	if err != nil {
		log.Fatal(err)
	}

	scan := bufio.NewScanner(input)

	visited := make(map[string]bool)

	var repoList RepoList

	for scan.Scan() {
		var line common.RepoLine

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

		repoList = append(repoList, line)
	}

	sort.Sort(&repoList)

	out, err := os.Create(*output)
	if err != nil {
		log.Fatal(err)
	}

	for i, line := range repoList {
		if i >= *limit {
			break
		}

		bytes, err := json.Marshal(line)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Fprintf(out, "%s\n", bytes)
	}
}
