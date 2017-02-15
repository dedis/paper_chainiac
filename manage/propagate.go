package manage

import (
	"sync"

	"time"

	"reflect"

	"gopkg.in/dedis/onet.v1"
	"gopkg.in/dedis/onet.v1/log"
	"gopkg.in/dedis/onet.v1/network"
)

func init() {
	onet.GlobalProtocolRegister("Propagate", NewPropagateProtocol)
	network.RegisterMessage(PropagateSendData{})
}

// Propagate is a protocol that sends some data to all attached nodes
// and waits for confirmation before returning.
type Propagate struct {
	*onet.TreeNodeInstance
	onData    func(network.Message)
	onDoneCb  func(int)
	sd        *PropagateSendData
	ChannelSD chan struct {
		*onet.TreeNode
		PropagateSendData
	}
	ChannelReply chan struct {
		*onet.TreeNode
		PropagateReply
	}

	received int
	subtree  int
	sync.Mutex
}

// PropagateSendData is the message to pass the data to the children
type PropagateSendData struct {
	// Data is the data to transmit
	Data []byte
	// How long the root will wait for the children before
	// timing out
	Msec int
}

// PropagateReply is sent from the children back to the root
type PropagateReply struct {
	Level int
}

// PropagateStartAndWait starts the propagation protocol and blocks until
// all children stored the new value or the timeout has been reached.
// The return value is the number of nodes that acknowledged having
// stored the new value or an error if the protocol couldn't start.
func PropagateStartAndWait(c *onet.Context, el *onet.Roster, msg network.Message, msec int, f func(network.Message)) (int, error) {
	tree := el.GenerateNaryTreeWithRoot(8, c.ServerIdentity())
	log.Lvl3("Starting to propagate", reflect.TypeOf(msg))
	pi, err := c.CreateProtocol("Propagate", tree)
	if err != nil {
		return -1, err
	}
	return propagateStartAndWait(pi, msg, msec, f)
}

// Separate function for testing
func propagateStartAndWait(pi onet.ProtocolInstance, msg network.Message, msec int, f func(network.Message)) (int, error) {
	d, err := network.Marshal(msg)
	if err != nil {
		return -1, err
	}
	protocol := pi.(*Propagate)
	protocol.Lock()
	protocol.sd.Data = d
	protocol.sd.Msec = msec
	protocol.onData = f

	done := make(chan int)
	protocol.onDoneCb = func(i int) { done <- i }
	protocol.Unlock()
	if err = protocol.Start(); err != nil {
		return -1, err
	}
	ret := <-done
	log.Lvl3("Finished propagation with", ret, "replies")
	return ret, nil
}

// NewPropagateProtocol returns a new Propagate protocol
func NewPropagateProtocol(n *onet.TreeNodeInstance) (onet.ProtocolInstance, error) {
	p := &Propagate{
		sd:               &PropagateSendData{[]byte{}, 120000},
		TreeNodeInstance: n,
		received:         0,
		subtree:          n.TreeNode().SubtreeCount(),
	}
	for _, h := range []interface{}{&p.ChannelSD, &p.ChannelReply} {
		if err := p.RegisterChannel(h); err != nil {
			return nil, err
		}
	}
	return p, nil
}

// Start will contact everyone and make the connections
func (p *Propagate) Start() error {
	log.Lvl4("going to contact", p.Root().ServerIdentity)
	p.SendTo(p.Root(), p.sd)
	return nil
}

// Dispatch can handle timeouts
func (p *Propagate) Dispatch() error {
	process := true
	log.Lvl4(p.ServerIdentity())
	for process {
		p.Lock()
		timeout := time.Millisecond * time.Duration(p.sd.Msec)
		p.Unlock()
		select {
		case msg := <-p.ChannelSD:
			log.Lvl3(p.ServerIdentity(), "Got data from", msg.ServerIdentity, "and setting timeout to", msg.Msec)
			p.sd.Msec = msg.Msec
			if !p.IsRoot() {
				log.Lvl3(p.ServerIdentity(), "Sending to parent")
				p.SendToParent(&PropagateReply{})
			}
			if p.IsLeaf() {
				process = false
			} else {
				log.Lvl3(p.ServerIdentity(), "Sending to children")
				p.SendToChildren(&msg.PropagateSendData)
			}
			if p.onData != nil {
				_, netMsg, err := network.Unmarshal(msg.Data)
				if err == nil {
					p.onData(netMsg)
				}
			}
		case <-p.ChannelReply:
			p.received++
			log.Lvl4(p.ServerIdentity(), "received:", p.received, p.subtree)
			if !p.IsRoot() {
				p.SendToParent(&PropagateReply{})
			}
			if p.received == p.subtree {
				process = false
			}
		case <-time.After(timeout):
			_, a, err := network.Unmarshal(p.sd.Data)
			log.Fatal("Timeout of %s reached. %v %s", timeout, a, err)
			process = false
		}
	}
	if p.IsRoot() {
		if p.onDoneCb != nil {
			p.onDoneCb(p.received + 1)
		}
	}
	p.Done()
	return nil
}

// RegisterOnDone takes a function that will be called once all connections
// are set up. The argument to the function is the number of children that
// sent OK after the propagation
func (p *Propagate) RegisterOnDone(fn func(int)) {
	p.onDoneCb = fn
}

// RegisterOnData takes a function that will be called once all connections
// are set up. The argument to the function is the number of children that
// sent OK after the propagation
func (p *Propagate) RegisterOnData(fn func(network.Message)) {
	p.onData = fn
}

// Config stores the basic configuration for that protocol.
func (p *Propagate) Config(d []byte, msec int) {
	p.sd.Data = d
	p.sd.Msec = msec
}
