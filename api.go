package debianupdate

import (
	"github.com/dedis/paper_17_usenixsec_chainiac/skipchain"
	"gopkg.in/dedis/onet.v1"
	"gopkg.in/dedis/onet.v1/log"
	"gopkg.in/dedis/onet.v1/network"
)

type Client struct {
	*onet.Client
	Roster *onet.Roster
	//ProjectID
	Root *network.ServerIdentity
}

func NewClient(r *onet.Roster) *Client {
	return &Client{
		Client: onet.NewClient(ServiceName),
		Roster: r,
		Root:   r.List[0],
	}
}

func (c *Client) LatestUpdates(latestIDs []skipchain.SkipBlockID) (*LatestBlocksRet,
	error) {
	lbs := &LatestBlocks{latestIDs}
	lbr := &LatestBlocksRetInternal{}
	err := c.SendProtobuf(c.Root, lbs, lbr)
	if err != nil {
		return nil, err
	}

	var updates [][]*skipchain.SkipBlock
	for _, l := range lbr.Lengths {
		updates = append(updates, lbr.Updates[0:l])
		lbr.Updates = lbr.Updates[l:]
	}
	return &LatestBlocksRet{lbr.Timestamp, updates}, nil
}

func (c *Client) LatestUpdatesForRepo(repoName string) (*LatestBlockRet, error) {
	lbs := &LatestBlockRepo{repoName}
	lbr := &LatestBlockRet{}
	err := c.SendProtobuf(c.Root, lbs, lbr)
	if err != nil {
		return nil, err
	}
	return lbr, nil
}

func (c *Client) TimestampRequests(names []string) (*TimestampRets, error) {
	t := &TimestampRequests{names}
	tr := &TimestampRets{}
	err := c.SendProtobuf(c.Root, t, tr)
	if err != nil {
		return nil, err
	}
	return tr, nil
}

func (c *Client) LatestRelease(repo string) (*LatestRelease, error) {

	// First we gather the latest skipblock
	lbr, err := c.LatestUpdatesForRepo(repo)
	if err != nil {
		return nil, err
	}

	// we extract the release
	_, r, err := network.Unmarshal(lbr.Update[0].Data)

	if err != nil {
		return nil, err
	}

	// from the release we extract the packages names + hashes + proofs
	release := r.(*Release)

	proofs := release.Proofs
	packages := release.Repository.Packages

	packageProofHash := map[string]PackageProof{}

	log.Lvl2("preparing the datas")

	for i, p := range packages {
		packageProofHash[p.Name] = PackageProof{p.Hash, proofs[i]}
	}

	// We need to return the root signed
	return &LatestRelease{release.RootID, packageProofHash, lbr.Update}, nil
}

func (c *Client) Rx() uint64 {
	return 0
}

func (c *Client) Tx() uint64 {
	return 0
}
