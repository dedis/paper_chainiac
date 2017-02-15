package main

import (
	"os"
	"sort"
	"time"

	"io/ioutil"

	"github.com/BurntSushi/toml"
	"github.com/dedis/paper_17_usenixsec_chainiac"
	"github.com/dedis/paper_17_usenixsec_chainiac/swupdate/service"
	"github.com/dedis/paper_17_usenixsec_chainiac/timestamp"
	"gopkg.in/dedis/onet.v1"
	"gopkg.in/dedis/onet.v1/app"
	"gopkg.in/dedis/onet.v1/log"
	"gopkg.in/dedis/onet.v1/simul/monitor"
)

func init() {
	onet.SimulationRegister("DebianUpdateOneClient", NewOneClientSimulation)
}

type oneClientSimulation struct {
	onet.SimulationBFTree
	Base                      int
	Height                    int
	NumberOfInstalledPackages int
	NumberOfPackagesInRepo    int
	Snapshots                 string // All the snapshots filenames
}

func NewOneClientSimulation(config string) (onet.Simulation, error) {
	es := &oneClientSimulation{Base: 2, Height: 10}
	_, err := toml.Decode(config, es)
	if err != nil {
		return nil, err
	}
	//log.SetDebugVisible(3)
	return es, nil
}

func (e *oneClientSimulation) Setup(dir string, hosts []string) (
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

	err = app.Copy(dir, "signedfile.deb")
	if err != nil {
		return nil, err
	}

	return sc, nil
}

func (e *oneClientSimulation) Run(config *onet.SimulationConfig) error {
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

	repos := make(map[string]*debianupdate.RepositoryChain)
	releases := make(map[string]*debianupdate.Release)

	updateClient := debianupdate.NewClient(config.Roster)

	var round *monitor.TimeMeasure

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
		// Store the repo, root and proofs in a release
		release := &debianupdate.Release{repo, root, proofs}

		// check if the skipchain has already been created for this repo
		sc, knownRepo := repos[repo.GetName()]

		if knownRepo {
			//round = monitor.NewTimeMeasure("add_to_skipchain")

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
			//round = monitor.NewTimeMeasure("create_skipchain")

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
	}
	log.Lvl2("Loading repository files - done")

	latest_release_update := monitor.NewTimeMeasure("client_receive_latest_release")
	bw_update := monitor.NewCounterIOMeasure("client_bw_debianupdate", updateClient)
	lr, err := updateClient.LatestRelease(e.Snapshots)
	if err != nil {
		log.Lvl1(err)
		return nil
	}
	bw_update.Record()
	latest_release_update.Record()

	// Check signature on root

	verify_sig := monitor.NewTimeMeasure("verify_signature")
	log.Lvl1("Verifying root signature")
	if err := lr.Update[0].VerifySignatures(); err != nil {
		log.Lvl1("Failed verification of root's signature")
		return err
	}
	verify_sig.Record()

	// Verify proofs for installed packages
	round = monitor.NewTimeMeasure("verify_proofs")

	// take e.NumberOfInstalledPackages randomly insteand of the first

	log.Lvl1("Verifying at most", e.NumberOfInstalledPackages, "packages")
	i := 1
	for name, p := range lr.Packages {
		hash := []byte(p.Hash)
		proof := p.Proof
		if proof.Check(debianupdate.HashFunc(), lr.RootID, hash) {
			log.Lvl2("Package", name, "correctly verified")
		} else {
			log.Fatal("The proof for " + name + " is not correct.")
		}
		i = i + 1
		if i > e.NumberOfInstalledPackages {
			break
		}
	}
	round.Record()

	// APT Emulation
	key := swupdate.NewPGP()
	signedFile, err := ioutil.ReadFile("signedfile.deb")
	log.ErrFatal(err)
	signature, err := key.Sign([]byte(signedFile))
	hashSample := "c8661d4926323ee7eba2db21f5d0c7978886137a0b8b122ca0231a408dc9ce08"
	round = monitor.NewTimeMeasure("apt_gpg_10times")
	for c := 0; c < 10; c++ {
		key.Verify([]byte(signedFile), signature)
		for i := 0; i < e.NumberOfInstalledPackages; i += 1 {
			if hashSample != "c8661d4926323ee7eba2db21f5d0c7978886137a0b8b122ca0231a408dc9ce08" {
				log.Fatal("should never happen")
			}
		}
		round.Record()
	}

	return nil
}
