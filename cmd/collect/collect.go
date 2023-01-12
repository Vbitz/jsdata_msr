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
	"runtime/pprof"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"eait.uq.edu.au/jscarsbrook/jsdata/v2/pkg/common"
	"eait.uq.edu.au/jscarsbrook/jsdata/v2/pkg/tsbridge"

	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/filesystem"

	"github.com/schollz/progressbar/v3"
)

var (
	repoList   = flag.String("repos", "repos.njson", "A newline delimited JSON file containing a list of repositories to download.")
	single     = flag.Bool("single", false, "Should a single repository be processed?")
	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
)

var counter = make(map[string]uint64)
var counterMtx sync.Mutex
var totalCommits uint64
var globalTsFiles uint64

func count(id string) {
	counterMtx.Lock()
	defer counterMtx.Unlock()

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
	PackageName       string
	PackageVersion    string
	TypeScriptVersion string
	Flags             FeatureFlags
}

type FeatureFlags struct {
	SatisfiesExpression                bool
	AccessorKeyword                    bool
	ExtendsConstraintOnInfer           bool
	VarianceAnnotationsOnTypeParameter bool
	TypeModifierOnImportName           bool
	ImportAssertion                    bool
	StaticBlockInClass                 bool
	OverrideOnClassMethod              bool
	AbstractConstructSignature         bool
	TemplateLiteralType                bool
	RemappedNameInMappedType           bool
	NamedTupleMember                   bool
	ShortCircuitAssignment             bool
}

func (f FeatureFlags) Merge(other *FeatureFlags) FeatureFlags {
	if other == nil {
		return f
	}
	return FeatureFlags{
		SatisfiesExpression:                f.SatisfiesExpression || other.SatisfiesExpression,
		AccessorKeyword:                    f.AccessorKeyword || other.AccessorKeyword,
		ExtendsConstraintOnInfer:           f.ExtendsConstraintOnInfer || other.ExtendsConstraintOnInfer,
		VarianceAnnotationsOnTypeParameter: f.VarianceAnnotationsOnTypeParameter || other.VarianceAnnotationsOnTypeParameter,
		TypeModifierOnImportName:           f.TypeModifierOnImportName || other.TypeModifierOnImportName,
		ImportAssertion:                    f.ImportAssertion || other.ImportAssertion,
		StaticBlockInClass:                 f.StaticBlockInClass || other.StaticBlockInClass,
		OverrideOnClassMethod:              f.OverrideOnClassMethod || other.OverrideOnClassMethod,
		AbstractConstructSignature:         f.AbstractConstructSignature || other.AbstractConstructSignature,
		TemplateLiteralType:                f.TemplateLiteralType || other.TemplateLiteralType,
		RemappedNameInMappedType:           f.RemappedNameInMappedType || other.RemappedNameInMappedType,
		NamedTupleMember:                   f.NamedTupleMember || other.NamedTupleMember,
		ShortCircuitAssignment:             f.ShortCircuitAssignment || other.ShortCircuitAssignment,
	}
}

func get(m map[string]bool, name string) bool {
	v, ok := m[name]
	if ok {
		return v
	} else {
		return false
	}
}

func GetFlagsFromResponse(resp tsbridge.Response) *FeatureFlags {
	return &FeatureFlags{
		SatisfiesExpression:                get(resp.Features, "SatisfiesExpression"),
		AccessorKeyword:                    get(resp.Features, "AccessorKeyword"),
		ExtendsConstraintOnInfer:           get(resp.Features, "ExtendsConstraintOnInfer"),
		VarianceAnnotationsOnTypeParameter: get(resp.Features, "VarianceAnnotationsOnTypeParameter"),
		TypeModifierOnImportName:           get(resp.Features, "TypeModifierOnImportName"),
		ImportAssertion:                    get(resp.Features, "ImportAssertion"),
		StaticBlockInClass:                 get(resp.Features, "StaticBlockInClass"),
		OverrideOnClassMethod:              get(resp.Features, "OverrideOnClassMethod"),
		AbstractConstructSignature:         get(resp.Features, "AbstractConstructSignature"),
		TemplateLiteralType:                get(resp.Features, "TemplateLiteralType"),
		RemappedNameInMappedType:           get(resp.Features, "RemappedNameInMappedType"),
		NamedTupleMember:                   get(resp.Features, "NamedTupleMember"),
		ShortCircuitAssignment:             get(resp.Features, "ShortCircuitAssignment"),
	}
}

var bridge = tsbridge.NewBridge("")

var ErrPackageJsonNotFound = fmt.Errorf("package.json not found")

func collectDataCommit(id string, repo *git.Repository, commit *object.Commit, visitedMap map[string]*FeatureFlags) (CommitData, error) {
	tree, err := commit.Tree()
	if err != nil {
		return CommitData{}, fmt.Errorf("error fetching tree: %v", err)
	}

	packageJsonFile, err := tree.File("package.json")
	if err != nil {
		return CommitData{}, ErrPackageJsonNotFound
	}

	pkgContents, err := packageJsonFile.Contents()
	if err != nil {
		return CommitData{}, fmt.Errorf("error getting package.json: %v", err)
	}

	packageJson, err := parsePackageJson(pkgContents)
	if err != nil {
		return CommitData{}, fmt.Errorf("error parsing package.json: %v", err)
	}

	tsVersion := getTsVersion(packageJson)

	visitBlob := func(blob *object.Blob) (*FeatureFlags, error) {
		atomic.AddUint64(&globalTsFiles, 1)

		reader, err := blob.Reader()
		if err != nil {
			return nil, err
		}
		defer reader.Close()

		content, err := io.ReadAll(reader)
		if err != nil {
			return nil, err
		}

		resp, err := bridge.Call(tsbridge.Request{
			Filename:     blob.Hash.String() + ".ts",
			FileContents: string(content),
		})
		if err != nil {
			log.Printf("err = %v", err)
			return nil, err
		}

		flags := GetFlagsFromResponse(resp)

		// log.Printf("resp = %+v", resp)

		return flags, nil
	}

	var visitTree func(tree *object.Tree) (*FeatureFlags, error)

	visitTree = func(tree *object.Tree) (*FeatureFlags, error) {
		retFlags := FeatureFlags{}

		for _, ent := range tree.Entries {
			if flags, ok := visitedMap[ent.Hash.String()]; ok {
				retFlags = retFlags.Merge(flags)
				continue
			}

			obj, err := repo.Object(plumbing.AnyObject, ent.Hash)
			if err == plumbing.ErrObjectNotFound {
				continue // Ignore these errors.
			} else if err != nil {
				return nil, err
			}

			var flags *FeatureFlags

			switch obj := obj.(type) {
			case *object.Blob:
				if strings.HasSuffix(ent.Name, ".ts") {
					flags, err = visitBlob(obj)
					if err != nil {
						return nil, err
					}
				}
			case *object.Tree:
				flags, err = visitTree(obj)
				if err != nil {
					return nil, err
				}
			default:
				continue
			}

			if flags != nil {
				visitedMap[ent.Hash.String()] = flags
				retFlags = retFlags.Merge(flags)
			} else {
				visitedMap[ent.Hash.String()] = nil
			}
		}

		return &retFlags, nil
	}

	flags, err := visitTree(tree)
	if err != nil {
		return CommitData{}, fmt.Errorf("error iterating tree: %v", err)
	}

	date := getCommitDate(commit)

	return CommitData{
		Id:                id,
		Date:              uint64(date.Unix()),
		Hash:              commit.Hash.String(),
		PackageName:       packageJson.Name,
		PackageVersion:    packageJson.Version,
		TypeScriptVersion: tsVersion,
		Flags:             *flags,
	}, nil
}

type RepoData struct {
	Id      string
	Commits []CommitData
}

type Commit struct {
	Hash plumbing.Hash
	Obj  *object.Commit
	When int64
}

type CommitList []Commit

// Len implements sort.Interface
func (lst *CommitList) Len() int {
	return len(*lst)
}

// Less implements sort.Interface
func (lst *CommitList) Less(i int, j int) bool {
	return (*lst)[i].When < (*lst)[j].When
}

// Swap implements sort.Interface
func (lst *CommitList) Swap(i int, j int) {
	(*lst)[i], (*lst)[j] = (*lst)[j], (*lst)[i]
}

func getCommitDate(commit *object.Commit) time.Time {
	authorDate := commit.Author.When
	commitDate := commit.Committer.When

	if authorDate.After(commitDate) {
		return authorDate
	} else {
		return commitDate
	}
}

func collectData(id string, repo *git.Repository) (RepoData, error) {
	var ret RepoData

	visitedMap := make(map[string]*FeatureFlags)

	var commitList CommitList

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

		date := getCommitDate(commit)

		// Exclude commits before 2020 or after 2022
		if date.Year() < 2020 || date.Year() > 2022 {
			continue
		}

		commitList = append(commitList, Commit{
			Hash: commit.Hash,
			When: date.Unix(),
			Obj:  commit,
		})
	}

	sort.Sort(&commitList)

	for _, commit := range commitList {
		atomic.AddUint64(&totalCommits, 1)

		commitData, err := collectDataCommit(id, repo, commit.Obj, visitedMap)
		if err == ErrPackageJsonNotFound {
			continue
		} else if err != nil {
			log.Printf("error in %s@%s: %v", id, commit.Hash.String()[:8], err)
			count("collectError")
			continue
		}

		ret.Commits = append(ret.Commits, commitData)
	}

	return ret, nil
}

func str(b bool) string {
	if b {
		return "1"
	} else {
		return "0"
	}
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

	visited := make(map[string]bool)

	rows := 0
	total := 0
	left := 0

	if *single {
		go func() {
			for {
				log.Printf("(%d tsFiles, %d commits)", globalTsFiles, totalCommits)
				time.Sleep(250 * time.Millisecond)
			}
		}()
	}

	prog := progressbar.Default(-1)

	var wg sync.WaitGroup

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

		wg.Add(1)
		total += 1
		left += 1

		go func() {
			defer func() {
				prog.Add(1)
				prog.Describe(fmt.Sprintf("(%d tsFiles, %d commits, %d/%d left)", globalTsFiles, totalCommits, left, total))
				left -= 1
				wg.Done()
			}()

			repo, err := cloneRepo(line.GitUrl)
			if err != nil {
				log.Printf("error opening: %v", err)
				return
			}

			count("opened")

			commitData, err := collectData(id, repo)
			if err != nil {
				count("failure")
				log.Printf("error collecting data %s/%s: %v", line.Login, line.Name, err)
				return
			}

			output, err := os.Create(path.Join("store", "output", line.Login+"_"+line.Name+".csv"))
			if err != nil {
				log.Fatal(err)
			}
			defer output.Close()

			csvOutput := csv.NewWriter(output)

			for _, commit := range commitData.Commits {
				rows += 1
				err := csvOutput.Write([]string{
					commit.Id,
					commit.Hash,
					commit.PackageName,
					commit.PackageVersion,
					commit.TypeScriptVersion,
					fmt.Sprintf("%d", commit.Date),
					str(commit.Flags.AccessorKeyword),
					str(commit.Flags.SatisfiesExpression),
					str(commit.Flags.ExtendsConstraintOnInfer),
					str(commit.Flags.VarianceAnnotationsOnTypeParameter),
					str(commit.Flags.TypeModifierOnImportName),
					str(commit.Flags.ImportAssertion),
					str(commit.Flags.StaticBlockInClass),
					str(commit.Flags.OverrideOnClassMethod),
					str(commit.Flags.AbstractConstructSignature),
					str(commit.Flags.TemplateLiteralType),
					str(commit.Flags.RemappedNameInMappedType),
					str(commit.Flags.NamedTupleMember),
					str(commit.Flags.ShortCircuitAssignment),
				})
				if err != nil {
					log.Fatal(err)
				}
			}

			csvOutput.Flush()

			count("success")
		}()
	}

	wg.Wait()

	for k, v := range counter {
		log.Printf("[C] %s = %d", k, v)
	}
}
