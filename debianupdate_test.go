package debianupdate

import (
	"flag"
	"os"
	"runtime/pprof"
	"strconv"
	"testing"
	"time"

	"github.com/dedis/simul/monitor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/dedis/onet.v1"
	"gopkg.in/dedis/onet.v1/crypto"
	"gopkg.in/dedis/onet.v1/log"
)

func init() {
	initGlobals()
}

func TestMain(m *testing.M) {
	os.RemoveAll("config")
	rc := map[string]string{}
	mon := monitor.NewMonitor(monitor.NewStats(rc))
	go func() { log.ErrFatal(mon.Listen()) }()
	local := "localhost:" + strconv.Itoa(monitor.DefaultSinkPort)
	log.ErrFatal(monitor.ConnectSink(local))

	flag.Parse()
	log.TestOutput(testing.Verbose(), 2)
	done := make(chan int)
	go func() {
		code := m.Run()
		done <- code
	}()
	select {
	case code := <-done:
		monitor.EndAndCleanup()
		log.AfterTest(nil)
		os.Exit(code)
	case <-time.After(log.MainTestWait):
		log.Error("Didn't finish in time")
		pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
		os.Exit(1)
	}
}

func TestDebianUpdate_RepositorySC(t *testing.T) {
	local := onet.NewLocalTest()
	defer local.CloseAll()

	_, roster, s := local.MakeHELS(5, debianUpdateService)
	service := s.(*DebianUpdate)

	release := chain1.blocks[0].release
	crr, err := service.CreateRepository(nil,
		&CreateRepository{roster, release, 2, 10})
	log.ErrFatal(err)
	repoChain := crr.(*CreateRepositoryRet).RepositoryChain

	reposcret, err := service.RepositorySC(nil, &RepositorySC{"unknown"})
	require.NotNil(t, err)
	reposcret, err = service.RepositorySC(nil,
		&RepositorySC{release.Repository.GetName()})
	log.ErrFatal(err)
	sc2 := reposcret.(*RepositorySCRet).Last

	require.Equal(t, repoChain.Data.Hash, sc2.Hash)
	sc3 := service.Storage.RepositoryChain[release.Repository.GetName()]
	require.Equal(t, repoChain.Data.Hash, sc3.Data.Hash)
}

func TestDebianUpdate_CreateRepository(t *testing.T) {
	local := onet.NewLocalTest()
	defer local.CloseAll()
	_, roster, s := local.MakeHELS(5, debianUpdateService)
	service := s.(*DebianUpdate)
	release1 := chain1.blocks[0].release
	rootHash := chain2.blocks[1].release.RootID
	repo1 := chain1.blocks[0].repo
	// This should fail as the signatures are wrong
	createRepo, err := service.CreateRepository(nil,
		&CreateRepository{
			Roster:  roster,
			Release: &Release{repo1, rootHash, release1.Proofs},
			Base:    2,
			Height:  10,
		})
	assert.NotNil(t, err, "Accepted wrong root")
	createRepo, err = service.CreateRepository(nil,
		&CreateRepository{roster, release1, 2, 10})
	log.ErrFatal(err)

	repoChain := createRepo.(*CreateRepositoryRet).RepositoryChain
	assert.NotNil(t, repoChain.Data)
	repo := repoChain.Release.Repository
	assert.Equal(t, *repo1, *repo)
	assert.Equal(t, *repo1,
		*service.Storage.RepositoryChain[repo.GetName()].Release.Repository)
	assert.Equal(t, release1.RootID,
		service.Storage.RepositoryChain[repo.GetName()].Release.RootID)
}

func TestDebianUpdate_UpdateRepository(t *testing.T) {
	local := onet.NewLocalTest()
	defer local.CloseAll()

	_, roster, s := local.MakeHELS(5, debianUpdateService)
	service := s.(*DebianUpdate)

	release1 := chain1.blocks[0].release
	release2 := chain1.blocks[1].release

	repo, err := service.CreateRepository(nil,
		&CreateRepository{roster, release1, 2, 10})
	log.ErrFatal(err)

	repoChain := repo.(*CreateRepositoryRet).RepositoryChain

	updateRepo, err := service.UpdateRepository(nil,
		&UpdateRepository{
			RepositoryChain: repoChain,
			Release:         release2,
		})
	log.ErrFatal(err)
	repoChain = updateRepo.(*UpdateRepositoryRet).RepositoryChain
	assert.NotNil(t, repoChain)
	assert.Equal(t, *chain1.blocks[1].repo, *repoChain.Release.Repository)
}

func TestDebianUpdate_PropagateBlock(t *testing.T) {
	local := onet.NewLocalTest()
	defer local.CloseAll()
	_, roster, s := local.MakeHELS(5, debianUpdateService)
	service := s.(*DebianUpdate)

	createRepo, err := service.CreateRepository(nil,
		&CreateRepository{roster, chain1.blocks[0].release, 2, 10})

	assert.Nil(t, err)
	log.ErrFatal(err)

	repoChain := createRepo.(*CreateRepositoryRet).RepositoryChain

	assert.Equal(t, repoChain.Release, chain1.blocks[0].release)
}

// repositoryChain tracks all test releases for one fake repo
type repositoryChain struct {
	repo   string
	blocks []*repositoryBlock
}

// packageBlock tracks all information on one release of a package
type repositoryBlock struct {
	repo    *Repository
	release *Release
}

var chain1 *repositoryChain
var chain2 *repositoryChain

func initGlobals() {
	createBlock := func(origin, suite, version string, packages []*Package) *repositoryBlock {
		repo := &Repository{
			Origin:   origin,
			Suite:    suite,
			Version:  version,
			Packages: packages,
			SourceUrl: "http://mirror.switch.ch/ftp/mirror/debian/dists/stable" +
				"/main/binary-amd64/",
		}

		hashes := make([]crypto.HashID, len(repo.Packages))
		for i, p := range repo.Packages {
			hashes[i] = crypto.HashID(p.Hash)
		}
		root, proofs := crypto.ProofTree(HashFunc(), hashes)
		return &repositoryBlock{
			repo:    repo,
			release: &Release{repo, root, proofs},
		}
	}

	createChain := func(origin string, suite string, packages []*Package) *repositoryChain {
		b1 := createBlock(origin, suite, "1.2", packages)
		b2 := createBlock(origin, suite, "1.3", packages)
		return &repositoryChain{
			repo:   origin + "-" + suite,
			blocks: []*repositoryBlock{b1, b2},
		}
	}
	packages1 := []*Package{
		{"test1", "0.1", "0000"},
		{"test2", "0.1", "0101"},
		{"test3", "0.1", "1010"},
		{"test4", "0.1", "1111"},
	}
	packages2 := []*Package{
		{"test1", "0.1", "000a"},
		{"test2", "0.1", "0101"},
		{"test3", "0.1", "1010"},
		{"test4", "0.1", "1111"},
	}
	chain1 = createChain("debian", "stable", packages1)
	chain2 = createChain("debian", "stable-update", packages2)
}
