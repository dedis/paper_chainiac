package timestamp

import (
	"errors"

	"time"

	"gopkg.in/dedis/onet.v1"
	"gopkg.in/dedis/onet.v1/log"
	"gopkg.in/dedis/onet.v1/network"
)

// Client is a structure to communicate with the CoSi
// service
type Client struct {
	*onet.Client
}

// NewClient instantiates a new Timestamp client
func NewClient() *Client {
	return &Client{Client: onet.NewClient(ServiceName)}
}

// SignMsg sends a CoSi sign request
func (c *Client) SignMsg(root *network.ServerIdentity, msg []byte) (*SignatureResponse, error) {
	serviceReq := &SignatureRequest{
		Message: msg,
	}
	log.Lvl4("Sending message [", string(msg), "] to", root)
	reply, err := c.Send(root, serviceReq)
	if e := onet.ErrMsg(reply, err); e != nil {
		return nil, e
	}
	sr, ok := reply.Msg.(SignatureResponse)
	if !ok {
		return nil, errors.New("This is odd: couldn't cast reply.")
	}
	return &sr, nil
}

// SetupStamper initializes the root node with the desired configuration
// parameters. The root node will start the main loop upon receiving this
// request.
// XXX This is a quick hack which simplifies the simulations.
func (c *Client) SetupStamper(roster *onet.Roster, epochDuration time.Duration,
	maxIterations int) (*SetupRosterResponse, error) {
	serviceReq := &SetupRosterRequest{
		Roster:        roster,
		EpochDuration: epochDuration,
		MaxIterations: maxIterations,
	}
	root := roster.List[0]
	log.Lvl4("Sending message to:", root)
	reply, err := c.Send(root, serviceReq)
	if e := onet.ErrMsg(reply, err); e != nil {
		return nil, e
	}
	sr, ok := reply.Msg.(SetupRosterResponse)
	if !ok {
		return nil, errors.New("This is odd: couldn't cast reply.")
	}
	log.Lvl4("Initialized timestamp with roster id:", sr.ID)
	return &sr, nil
}
