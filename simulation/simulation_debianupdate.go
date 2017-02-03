package main

import (
	"os"
	"sort"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/dedis/paper_17_usenixsec_chainiac"
	"github.com/dedis/paper_17_usenixsec_chainiac/timestamp"
	"gopkg.in/dedis/onet.v1"
	"gopkg.in/dedis/onet.v1/log"
	"gopkg.in/dedis/onet.v1/simul/monitor"
)

func init() {
	onet.SimulationRegister("DebianUpdateCreate", NewCreateSimulation)
}

type createSimulation struct {
	onet.SimulationBFTree
	Base                   int
	Height                 int
	NumberOfPackagesInRepo int
	Snapshots              string // All the snapshots filenames
}

// NewCreateSimulation returns the new simulation where all fields are
// initialized using the config-file
func NewCreateSimulation(config string) (onet.Simulation, error) {
	es := &createSimulation{Base: 2, Height: 10}
	_, err := toml.Decode(config, es)
	if err != nil {
		return nil, err
	}
	//log.SetDebugVisible(3)
	return es, nil
}

// Setup creates the tree used for that simulation (cothorities and link
// between them)
func (e *createSimulation) Setup(dir string, hosts []string) (
	*onet.SimulationConfig, error) {

	sc := &onet.SimulationConfig{}
	e.CreateRoster(sc, hosts, 2000)
	err := e.CreateTree(sc)

	if err != nil {
		return nil, err
	}
	err = CopyDir(dir, e.Snapshots)
	if err != nil {
		return nil, err
	}
	return sc, nil
}

// Run is used on the destination machines and runs a number of rounds.
func (e *createSimulation) Run(config *onet.SimulationConfig) error {

	// The cothority configuration
	size := config.Tree.Size()
	log.Lvl2("Size is:", size, "rounds:", e.Rounds)

	// check if the service is running and get an handle to it
	service, ok := config.GetService(debianupdate.ServiceName).(*debianupdate.DebianUpdate)
	if service == nil || !ok {
		log.Fatal("Didn't find service", debianupdate.ServiceName)
	}

	// create and setup a new timestamp client
	c := timestamp.NewClient()
	maxIterations := 0
	_, err := c.SetupStamper(config.Roster, time.Millisecond*250, maxIterations)
	if err != nil {
		return nil
	}

	// get the release and snapshots
	current_dir, err := os.Getwd()
	if err != nil {
		return nil
	}
	snapshot_files, err := GetFileFromType(current_dir+"/"+e.Snapshots, "Packages")
	if err != nil {
		return nil
	}
	release_files, err := GetFileFromType(current_dir+"/"+e.Snapshots, "Release")
	if err != nil {
		return nil
	}

	sort.Sort(snapshot_files)
	sort.Sort(release_files)

	// Map a repo name to a skipchain
	repos := make(map[string]*debianupdate.RepositoryChain)
	// Map a repo name to a release (which is the repo + the signed root + proof)
	releases := make(map[string]*debianupdate.Release)

	log.Lvl2("Loading repository files")
	for i, release_file := range release_files {
		log.Lvl1("Parsing repo file", release_file)

		// Create a new repository structure (not a new skipchain..!)
		repo, err := debianupdate.NewRepository(release_file, snapshot_files[i],
			"https://snapshots.debian.org", e.Snapshots, e.NumberOfPackagesInRepo)
		log.ErrFatal(err)
		log.Lvl1("Repository created with", len(repo.Packages), "packages")

		// Recover all the hashes from the repo
		hashes := make([]timestamp.HashID, len(repo.Packages))
		for i, p := range repo.Packages {
			hashes[i] = timestamp.HashID(p.Hash)
		}

		// Compute the root and the proofs
		root, proofs := timestamp.ProofTree(debianupdate.HashFunc(), hashes)
		lengths := []int64{}
		for _, proof := range proofs {
			lengths = append(lengths, int64(len(proof.Proof)))
		}
		// Store the repo, root and proofs in a release
		release := &debianupdate.Release{repo, root, proofs, lengths}

		// check if the skipchain has already been created for this repo
		sc, knownRepo := repos[repo.GetName()]

		var round *monitor.TimeMeasure
		if knownRepo {
			round = monitor.NewTimeMeasure("add_to_skipchain")

			log.Lvl1("A skipchain for", repo.GetName(), "already exists",
				"trying to add the release to the skipchain.")

			// is the new block different ?
			// who should take care of that ? the client or the server ?
			// I would say the server, when it receive a new release
			// it should check that it's different than the actual release
			urr, err := service.UpdateRepository(
				&debianupdate.UpdateRepository{sc, release})

			if err != nil {
				log.Lvl1(err)
			} else {

				// update the references to the latest block and release
				repos[repo.GetName()] = urr.(*debianupdate.UpdateRepositoryRet).RepositoryChain
				releases[repo.GetName()] = release
			}
		} else {
			round = monitor.NewTimeMeasure("create_skipchain")

			log.Lvl2("Creating a new skipchain for", repo.GetName())

			cr, err := service.CreateRepository(
				&debianupdate.CreateRepository{config.Roster, release, e.Base, e.Height})
			if err != nil {
				return err
			}

			// update the references to the latest block and release
			repos[repo.GetName()] = cr.(*debianupdate.CreateRepositoryRet).RepositoryChain
			releases[repo.GetName()] = release
		}
		round.Record()
	}
	log.Lvl2("Loading repository files - done")
	return nil
}
