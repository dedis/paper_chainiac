package debianupdate

import (
	"github.com/dedis/paper_17_usenixsec_chainiac_cleanup/skipchain"
	"github.com/dedis/paper_17_usenixsec_chainiac_cleanup/timestamp"
	"github.com/satori/go.uuid"
	"gopkg.in/dedis/onet.v1"
	"gopkg.in/dedis/onet.v1/crypto"
	"gopkg.in/dedis/onet.v1/network"
)

func init() {
	for _, msg := range []interface{}{
		RepositoryChain{},
		CreateRepository{},
		CreateRepositoryRet{},
		UpdateRepository{},
		UpdateRepositoryRet{},
		Release{},
		Repository{},
		LatestBlocks{},
		LatestBlocksRet{},
		LatestBlocksRetInternal{},
		TimestampRets{},
		LatestBlockRepo{},
		LatestBlockRet{},
		LatestRelease{},
		PackageProof{},
	} {
		network.RegisterMessage(msg)
	}
}

type ProjectID uuid.UUID

// Release is a Debian Repository and the developers' signatures
type Release struct {
	Repository   *Repository
	RootID       crypto.HashID
	Proofs       []crypto.Proof
	ProofsLength []int64
}

type RepositoryChain struct {
	Root    *skipchain.SkipBlock // The Root Skipchain
	Data    *skipchain.SkipBlock // The Data Skipchain
	Release *Release             // The Release (Repository) informations
}

type Timestamp struct {
	timestamp.SignatureResponse
	Proofs []crypto.Proof
}

type CreateRepository struct {
	Roster  *onet.Roster
	Release *Release
	Base    int
	Height  int
}

type CreateRepositoryRet struct {
	RepositoryChain *RepositoryChain
}

type UpdateRepository struct {
	RepositoryChain *RepositoryChain
	Release         *Release
}

type UpdateRepositoryRet struct {
	RepositoryChain *RepositoryChain
}

type RepositorySC struct {
	repositoryName string
}

// If no skipchain for PackageName is found, both first and last are nil.
// If the skipchain has been found, both the genesis-block and the latest
// block will be returned.
type RepositorySCRet struct {
	First *skipchain.SkipBlock
	Last  *skipchain.SkipBlock
}

// Request skipblocks needed to get to the latest version of the repository.
// LastKnownSB is the latest skipblock known to the client.
type LatestBlock struct {
	LastKnownSB skipchain.SkipBlockID
}

// Similar to LatestBlock but asking update information for all blocks being
// managed by the service.
type LatestBlocks struct {
	LastKnownSBs []skipchain.SkipBlockID
}

type LatestBlockRepo struct {
	RepoName string
}

// Returns the timestamp of the latest skipblock, together with an eventual
// shortes-link of skipblocks needed to go from the LastKnownSB to the
// current skipblock.
type LatestBlockRet struct {
	Timestamp *Timestamp
	Update    []*skipchain.SkipBlock
}

// Internal structure with lengths
type LatestBlocksRetInternal struct {
	Timestamp *Timestamp
	// Each updates for each repository ordered in same order that in LatestBlocks
	Updates []*skipchain.SkipBlock
	// STUPID: [][] is not correctly parsed by protobuf, so use lengths...
	Lengths []int64
}

// Similar to LatestBlockRet but gives information on *all* repository
type LatestBlocksRet struct {
	Timestamp *Timestamp
	// Each updates for each packages ordered in same order that in LatestBlocks
	Updates [][]*skipchain.SkipBlock
}

// TimestampRequest asks the debianupdate service to give back the proof of
// inclusion for the latest timestamp merkle tree including the repository
// denoted by Name.
type TimestampRequest struct {
	Name string
}

// Similar to TimestampRequest but asking more multiple proof at the same time
type TimestampRequests struct {
	Names []string
}

// Returns the Proofs to use to verify the inclusion of the repository given in
// TimestampRequest
type TimestampRet struct {
	Proof crypto.Proof
}

// Similar to TimestampRet but returns the requested proofs designated by
// repositories names.
type TimestampRets struct {
	Proofs map[string]crypto.Proof
}

type PackageProof struct {
	Hash  string
	Proof crypto.Proof
}

type LatestRelease struct {
	RootID   crypto.HashID
	Packages map[string]PackageProof
	Update   []*skipchain.SkipBlock
}
