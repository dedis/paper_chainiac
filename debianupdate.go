package debianupdate

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/dedis/paper_17_usenixsec_chainiac_cleanup/manage"
	"github.com/dedis/paper_17_usenixsec_chainiac_cleanup/skipchain"
	"github.com/dedis/paper_17_usenixsec_chainiac_cleanup/swupdate"
	"github.com/dedis/paper_17_usenixsec_chainiac_cleanup/timestamp"
	"github.com/satori/go.uuid"
	"gopkg.in/dedis/onet.v1"
	"gopkg.in/dedis/onet.v1/log"
	"gopkg.in/dedis/onet.v1/network"
	"gopkg.in/dedis/onet.v1/simul/monitor"
)

// ServiceName is the name to refer to the CoSi service
const ServiceName = "DebianUpdate"

var debianUpdateService onet.ServiceID

var verifierID = skipchain.VerifierID(uuid.NewV5(uuid.NamespaceURL, ServiceName))

func init() {
	onet.RegisterNewService(ServiceName, NewDebianUpdate)
	debianUpdateService = onet.ServiceFactory.ServiceID(ServiceName)
	network.RegisterMessage(&storage{})
	skipchain.VerificationRegistration(verifierID, verifierFunc)
}

// DebianUpdate service
type DebianUpdate struct {
	*onet.ServiceProcessor
	path           string
	Storage        *storage
	skipchain      *skipchain.Client
	ReasonableTime time.Duration
	sync.Mutex
}

type storage struct {
	Timestamp *Timestamp
	// the first block of the *string* repo
	RepositoryChainGenesis map[string]*RepositoryChain
	// the latest block
	RepositoryChain map[string]*RepositoryChain
	// the root skipchain
	Root *skipchain.SkipBlock
	// the interval between Timestamps
	TSInterval time.Duration
}

func NewDebianUpdate(context *onet.Context, path string) onet.Service {
	service := &DebianUpdate{
		ServiceProcessor: onet.NewServiceProcessor(context),
		path:             path,
		skipchain:        skipchain.NewClient(),
		Storage: &storage{
			RepositoryChainGenesis: map[string]*RepositoryChain{},
			RepositoryChain:        map[string]*RepositoryChain{},
		},
		ReasonableTime: time.Hour,
	}

	err := service.RegisterHandlers(service.CreateRepository,
		service.UpdateRepository, service.LatestBlocks,
		service.LatestBlockFromName, service.LatestBlock)
	/*service.TimeStampProof,
	service.LatestBlocks,
	service.TimestampProofs)*/

	if err != nil {
		log.ErrFatal(err, "Couldn't register messages")
	}

	return service
}

func (service *DebianUpdate) CreateRepository(si *network.ServerIdentity,
	cr *CreateRepository) (network.Message, error) {
	repo := cr.Release.Repository
	log.Lvlf3("%s Creating repository %s version %s", service,
		repo.GetName(), repo.Version)

	repoChain := &RepositoryChain{
		Release: cr.Release,
		Root:    service.Storage.Root,
	}
	if service.Storage.Root == nil {
		log.Lvl3("Creating Root-skipchain")
		var err error
		service.Storage.Root, err = service.skipchain.CreateRoster(cr.Roster,
			cr.Base, cr.Height, skipchain.VerifyNone, nil)
		if err != nil {
			return nil, err
		}
		repoChain.Root = service.Storage.Root
	}
	log.Lvl3("Creating Data-skipchain")
	var err error
	repoChain.Root, repoChain.Data, err = service.skipchain.CreateData(
		repoChain.Root, cr.Base, cr.Height, verifierID, cr.Release)
	if err != nil {
		log.Lvl2("error while adding the data in the skipchain")
		return nil, err
	}
	service.Storage.RepositoryChainGenesis[repo.GetName()] = repoChain
	if err := service.startPropagate(repo.GetName(), repoChain); err != nil {
		return nil, err
	}
	service.timestamp(time.Now())

	return &CreateRepositoryRet{repoChain}, nil
}

func (service *DebianUpdate) startPropagate(repo string,
	repoChain *RepositoryChain) error {
	roster := service.Storage.Root.Roster
	log.Lvl2("Propagating repository", repo, "to", roster.List)
	replies, err := manage.PropagateStartAndWait(service.Context, roster,
		repoChain, 120000, service.PropagateSkipBlock)
	if err != nil {
		return err
	}
	if replies != len(roster.List) {
		log.Warn("Did only get", replies, "out of", len(roster.List))
	}
	return nil
}

func (service *DebianUpdate) PropagateSkipBlock(msg network.Message) {
	repoChain, ok := msg.(*RepositoryChain)
	if !ok {
		log.Error("Couldn't convert to SkipBlock")
		return
	}
	repo := repoChain.Release.Repository.GetName()
	log.Lvl2("saving repositorychain for", repo)
	// TODO: Verification
	if _, exists := service.Storage.RepositoryChainGenesis[repo]; !exists {
		service.Storage.RepositoryChainGenesis[repo] = repoChain
	}
	service.Storage.RepositoryChain[repo] = repoChain
}

// timestamp creates a merkle tree of all the latests skipblocks of each
// skipchains, run a timestamp protocol and store the results in
// service.latestTimestamps.
func (service *DebianUpdate) timestamp(time time.Time) {
	//measure := monitor.NewTimeMeasure("debianupdate_timestamp")
	// order all packets and marshal them
	ids := service.orderedLatestSkipblocksID()
	// create merkle tree + proofs and the final message
	root, proofs := timestamp.ProofTree(HashFunc(), ids)
	msg := MarshalPair(root, time.Unix())
	// run protocol
	signature := service.cosiSign(msg)
	// TODO XXX Here in a non-academical world we should test if the
	// signature contains enough participants.
	service.updateTimestampInfo(root, proofs, time.Unix(), signature)
	//measure.Record()
}

func (service *DebianUpdate) cosiSign(msg []byte) []byte {
	sdaTree := service.Storage.Root.Roster.GenerateBinaryTree()

	// TODO XXX Here we use the swupdate protocol (should we ?)
	tni := service.NewTreeNodeInstance(sdaTree, sdaTree.Root, swupdate.ProtocolName)
	pi, err := swupdate.NewCoSiUpdate(tni, service.cosiVerify)
	if err != nil {
		panic("Couldn't make new protocol: " + err.Error())
	}
	service.RegisterProtocolInstance(pi)

	// measure the time the cothority takes to sign the root
	measure := monitor.NewTimeMeasure("cothority_signing")

	pi.SigningMessage(msg)
	// Take the raw message (already expecting a hash for the timestamp service)
	response := make(chan []byte)
	pi.RegisterSignatureHook(func(sig []byte) {
		response <- sig
	})
	go pi.Dispatch()
	go pi.Start()
	log.Lvl2("Waiting on cosi response ...")
	res := <-response
	measure.Record()
	log.Lvl2("... DONE: Recieved cosi response")
	return res
}

func (service *DebianUpdate) cosiVerify(msg []byte) bool {
	signedRoot, signedTime := UnmarshalPair(msg)
	// check timestamp
	if time.Now().Sub(time.Unix(signedTime, 0)) > service.ReasonableTime {
		log.Lvl2("Timestamp is too far in the past")
		return false
	}
	// check merkle tree root
	// order all packets and marshal them
	ids := service.orderedLatestSkipblocksID()

	// create merkle tree + proofs and the final message

	root, _ := timestamp.ProofTree(HashFunc(), ids)

	// root of merkle tree is not secret, no need to use constant time.
	if !bytes.Equal(root, signedRoot) {
		log.Lvl2("Root of merkle root does not match")
		return false
	}

	log.Lvl3("DebianUpdate cosi signature verified")
	return true
}

func (service *DebianUpdate) updateTimestampInfo(rootID timestamp.HashID,
	proofs []timestamp.Proof, ts int64, sig []byte) {
	service.Lock()
	defer service.Unlock()
	if service.Storage.Timestamp == nil {
		service.Storage.Timestamp = &Timestamp{}
	}
	var t = service.Storage.Timestamp
	t.Timestamp = ts
	t.Root = rootID
	t.Signature = sig
	t.Proofs = proofs
}

// HashFunc used for the timestamp operations with the Merkle tree generation
// and verification.
func HashFunc() timestamp.HashFunc {
	return sha256.New
}

// MarshalPair takes the root of a merkle tree (only a slice of bytes) and a
// unix timestamp and marshal them. UnmarshalPair do the opposite.
func MarshalPair(root timestamp.HashID, time int64) []byte {
	var buff bytes.Buffer
	if err := binary.Write(&buff, binary.BigEndian, time); err != nil {
		panic(err)
	}
	return append(buff.Bytes(), []byte(root)...)
}

// UnmarshalPair takes a slice of bytes generated by MarshalPair and retrieve
// the root and the unix timestamp out of it.
func UnmarshalPair(buff []byte) (timestamp.HashID, int64) {
	var reader = bytes.NewBuffer(buff)
	var time int64
	if err := binary.Read(reader, binary.BigEndian, &time); err != nil {
		panic(err)
	}
	return reader.Bytes(), time
}

// orderedLatestSkipblocksID sorts the latests blocks of all skipchains and
// return all ids in an array of HashID
func (service *DebianUpdate) orderedLatestSkipblocksID() []timestamp.HashID {
	keys := service.getOrderedRepositoryNames()

	ids := make([]timestamp.HashID, 0)
	chains := service.Storage.RepositoryChain
	for _, key := range keys {
		ids = append(ids, timestamp.HashID(chains[key].Data.Hash))
	}
	return ids
}

func (service *DebianUpdate) getOrderedRepositoryNames() []string {
	keys := make([]string, 0)
	for k := range service.Storage.RepositoryChain {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (service *DebianUpdate) UpdateRepository(si *network.ServerIdentity,
	ur *UpdateRepository) (network.Message, error) {
	//addBlock := monitor.NewTimeMeasure("add_block")
	//defer addBlock.Record()

	repoChain := &RepositoryChain{
		Release: ur.Release,
		Root:    ur.RepositoryChain.Root,
	}
	release := ur.Release

	actualRoot := service.Storage.RepositoryChain[release.Repository.GetName()].
		Release.RootID
	// Check if the new block is different
	if !bytes.Equal(actualRoot, release.RootID) {

		log.Lvl1("Adding new data to the Data-skipchain")
		ret, err := service.skipchain.ProposeData(ur.RepositoryChain.Root,
			ur.RepositoryChain.Data, release)
		if err != nil {
			return nil, err
		}
		repoChain.Data = ret.Latest

		if err := service.startPropagate(release.Repository.GetName(),
			repoChain); err != nil {
			return nil, err
		}
	} else {
		log.Lvl1("The latest existing skipblock is the same," +
			" only update the timestamp.")
		repoChain = service.Storage.RepositoryChain[release.Repository.GetName()]
	}

	service.timestamp(time.Now())
	return &UpdateRepositoryRet{repoChain}, nil
}

// NewProtocol initialize the Protocol
func (service *DebianUpdate) NewProtocol(tn *onet.TreeNodeInstance,
	conf *onet.GenericConfig) (onet.ProtocolInstance, error) {
	var pi onet.ProtocolInstance
	var err error
	switch tn.ProtocolName() {
	case "Propagate":
		log.Lvl2("DebianUpdate Service received New Protocol PROPAGATE event")
		pi, err = manage.NewPropagateProtocol(tn)
		if err != nil {
			return nil, err
		}
		pi.(*manage.Propagate).RegisterOnData(service.PropagateSkipBlock)
	default:
		log.Lvl2("DebianUpdate Service received New Protocol COSI event")
		pi, err = swupdate.NewCoSiUpdate(tn, service.cosiVerify)
		if err != nil {
			return nil, err
		}
	}
	return pi, err
}

func verifierFunc(msg, data []byte) bool {
	_, sbBuf, err := network.Unmarshal(data)
	sb, ok := sbBuf.(*skipchain.SkipBlock)
	if err != nil || !ok {
		log.Error(err, ok)
		return false
	}
	_, relBuf, err := network.Unmarshal(sb.Data)
	release, ok := relBuf.(*Release)
	if err != nil || !ok {
		log.Error(err, ok)
		return false
	}
	repo := release.Repository
	if repo == nil {
		log.Lvl2("The repository contained in the release is nil")
		return false
	}
	root := release.RootID
	if len(root) == 0 {
		log.Lvl2("No root hash, has the Merkle-tree correctly been built ?")
		return false
	}

	//ver := monitor.NewTimeMeasure("verification")

	// build the merkle-tree for packages
	hashes := make([]timestamp.HashID, len(repo.Packages))
	for i, p := range repo.Packages {
		hashes[i] = timestamp.HashID(p.Hash)
	}

	// measure the time the cothority takes to verify the merkle tree
	measure := monitor.NewTimeMeasure("cothority_verify_proofs")

	possibleRoot, _ := timestamp.ProofTree(HashFunc(), hashes)

	if !bytes.Equal(possibleRoot, root) {
		log.Lvl2("Wrong root hash")
		return false
	}

	measure.Record()

	return true
}

func (service *DebianUpdate) RepositorySC(si *network.ServerIdentity,
	rsc *RepositorySC) (network.Message, error) {

	repoChain, ok := service.Storage.RepositoryChain[rsc.repositoryName]

	if !ok {
		return nil, errors.New("Does not exist.")
	}

	latestBlockRet, err := service.LatestBlock(nil,
		&LatestBlock{repoChain.Data.Hash})

	if err != nil {
		return nil, err
	}

	update := latestBlockRet.(*LatestBlockRet).Update
	return &RepositorySCRet{
		First: service.Storage.RepositoryChainGenesis[rsc.repositoryName].Data,
		Last:  update[len(update)-1],
	}, nil
}

func (service *DebianUpdate) LatestBlock(si *network.ServerIdentity,
	lb *LatestBlock) (network.Message, error) {

	if service.Storage.Timestamp == nil {
		return nil, errors.New("Timestamp-service missing!")
	}

	gucRet, err := service.skipchain.GetUpdateChain(service.Storage.Root,
		lb.LastKnownSB)

	if err != nil {
		return nil, err
	}

	return &LatestBlockRet{service.Storage.Timestamp, gucRet.Update}, nil
}

func (service *DebianUpdate) LatestBlockFromName(si *network.ServerIdentity,
	lbr *LatestBlockRepo) (network.Message, error) {
	repoName := lbr.RepoName

	chain := service.Storage.RepositoryChain[repoName]
	if chain == nil {
		return nil, errors.New("skipchain not found for " + repoName)
	}
	return service.LatestBlock(nil, &LatestBlock{chain.Data.Hash})
}

func (service *DebianUpdate) LatestBlocks(si *network.ServerIdentity,
	lbs *LatestBlocks) (network.Message, error) {
	var updates []*skipchain.SkipBlock
	var lengths []int64
	var t *Timestamp
	for _, id := range lbs.LastKnownSBs {
		b, err := service.LatestBlock(nil, &LatestBlock{id})
		if err != nil {
			return nil, err
		}
		lb := b.(*LatestBlockRet)
		if len(lb.Update) > 1 {
			updates = append(updates, lb.Update...)
			lengths = append(lengths, int64(len(lb.Update)))
			if t == nil {
				t = lb.Timestamp
			}
		}
	}
	return &LatestBlocksRetInternal{t, updates, lengths}, nil
}
