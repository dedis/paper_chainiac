package debianupdate

import (
	"gopkg.in/dedis/onet.v1/log"

	"bufio"
	"compress/gzip"
	"io/ioutil"
	"os"
	"sort"
	"strings"
	//"sync"

	"regexp"
)

/*
 * Implement a Debian Repository containing packages
 */

type Repository struct {
	Origin    string
	Suite     string
	Version   string
	Packages  PackageSlice
	SourceUrl string
	//sync.Mutex
}

// NewRepository create a new repository from a release file, a packages file
// a keys file and a source url
func NewRepository(releaseFile string, packagesFile string,
	sourceUrl string, dir string, maxPackages int) (*Repository, error) {

	release, err := ioutil.ReadFile(dir + "/" + releaseFile)
	log.ErrFatal(err)

	repository := &Repository{SourceUrl: sourceUrl}

	for _, line := range strings.Split(string(release), "\n") {

		if strings.Contains(line, "Origin:") {
			repository.Origin = strings.Replace(line, "Origin: ", "", 1)
		} else if strings.Contains(line, "Archive:") {
			repository.Suite = strings.Replace(line, "Archive: ", "", 1)
		} else if strings.Contains(line, "Version:") {
			repository.Version = strings.Replace(line, "Version: ", "", 1)
		}
	}
	file_p, err := os.Open(dir + "/" + packagesFile)
	if err != nil {
		log.Fatal(err)
	}
	defer file_p.Close()
	gr, err := gzip.NewReader(file_p)
	if err != nil {
		log.Fatal(err)
	}
	defer gr.Close()

	scanner := bufio.NewScanner(gr)
	log.ErrFatal(err)

	packageString := ""

	// only import the maxPackages first packages
	i := 0
	for scanner.Scan() {
		if i >= maxPackages {
			break
		}
		line := scanner.Text()

		if line != " " && line != "" && line != "\n" {
			packageString += line + "\n"
		} else {
			// TODO go repository.AddPackage(packageString) with chan instead of mutex
			repository.AddPackage(packageString)
			packageString = ""
			i = i + 1
		}

	}

	if len(packageString) != 0 {
		repository.AddPackage(packageString)
		packageString = ""
	}

	sort.Sort(repository.Packages)

	/*for _, p := range repository.Packages {
		log.Print(p)
	}*/

	return repository, nil
}

// NewRepository create a new repository from a release file, a packages file
// a keys file and a source url
func NewRepositoryPopularity(releaseFile, packagesFile, popularFile,
	sourceUrl, dir string, maxPackages int) (*Repository, error) {

	release, err := ioutil.ReadFile(dir + "/" + releaseFile)
	log.ErrFatal(err)

	repository := &Repository{SourceUrl: sourceUrl}

	for _, line := range strings.Split(string(release), "\n") {

		if strings.Contains(line, "Origin:") {
			repository.Origin = strings.Replace(line, "Origin: ", "", 1)
		} else if strings.Contains(line, "Archive:") {
			repository.Suite = strings.Replace(line, "Archive: ", "", 1)
		} else if strings.Contains(line, "Version:") {
			repository.Version = strings.Replace(line, "Version: ", "", 1)
		}
	}
	file_p, err := os.Open(dir + "/" + packagesFile)
	log.ErrFatal(err)
	defer file_p.Close()
	gr, err := gzip.NewReader(file_p)
	log.ErrFatal(err)
	defer gr.Close()

	scanner := bufio.NewScanner(gr)
	log.ErrFatal(err)

	packageString := ""

	popular, err := ioutil.ReadFile(popularFile)
	log.ErrFatal(err)
	fetchPackages := make([]string, maxPackages)
	re := regexp.MustCompile(".*? +(.*?) .*")
	for i, s := range strings.Split(string(popular), "\n")[11 : maxPackages+11] {
		fetchPackages[i] = re.ReplaceAllString(s, "$1")
	}

	// only import the maxPackages first packages
	i := 0
	for scanner.Scan() {
		if i >= maxPackages {
			break
		}
		line := scanner.Text()

		if line != " " && line != "" && line != "\n" {
			packageString += line + "\n"
		} else {
			// TODO go repository.AddPackage(packageString) with chan instead of mutex
			repository.AddPackage(packageString)
			packageString = ""
			i = i + 1
		}

	}

	if len(packageString) != 0 {
		repository.AddPackage(packageString)
		packageString = ""
	}

	sort.Sort(repository.Packages)

	/*for _, p := range repository.Packages {
		log.Print(p)
	}*/

	return repository, nil
}

func (r *Repository) AddPackage(packageString string) {
	//r.Lock()
	//defer r.Unlock()
	p, err := NewPackage(packageString)
	log.ErrFatal(err)
	r.Packages = append(r.Packages, p)
}

func (r *Repository) GetName() string {
	return r.Origin + "-" + r.Suite
}
