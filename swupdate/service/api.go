package swupdate

import (
	"github.com/dedis/paper_17_usenixsec_chainiac/skipchain"
	"gopkg.in/dedis/onet.v1"
	"gopkg.in/dedis/onet.v1/network"
)

// Client is a structure to communicate with the software-update service.
type Client struct {
	*onet.Client
	Roster *onet.Roster
	ProjectID
	Root *network.ServerIdentity
}

// NewClient instantiates a new communication with the swupdate-client.
func NewClient(r *onet.Roster) *Client {
	return &Client{
		Client: onet.NewClient(ServiceName),
		Roster: r,
		Root:   r.List[0],
	}
}

func (c *Client) LatestUpdates(latestIDs []skipchain.SkipBlockID) (*LatestBlocksRet, error) {
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

func (c *Client) TimestampRequests(names []string) (*TimestampRets, error) {
	t := &TimestampRequests{names}
	tr := &TimestampRets{}
	err := c.SendProtobuf(c.Root, t, tr)
	if err != nil {
		return nil, err
	}
	return tr, nil
}

// Hack for paper
func (c *Client) Rx() uint64 {
	return 0
}
func (c *Client) Tx() uint64 {
	return 0
}
