package p2p

import (
	"context"
	"crypto/rand"
	"fmt"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	"github.com/multiformats/go-multiaddr"
	"log"
	"time"
)

// Instance is for creating and using a libp2p peer
type Instance struct {
	Port       int
	GroupName  string
	Host       *host.Host
	DHT        *dht.IpfsDHT
	PeerNotify *PeerNotify
	StartMdns  func() error
	CloseMdns  func() error
}

// PeerNotify is a handle for receiving mDNS peer discovery notifications
type PeerNotify struct {
	C chan peer.AddrInfo
}

// NewLibP2PInstance initialise a libp2p host, and use this host to initialise a DHT
func NewLibP2PInstance(p2pPort int, groupName string, h *host.Host) *Instance {
	var p2pHost *host.Host
	if h != nil {
		p2pHost = h
	} else {
		// Creates a new RSA key pair for this host
		r := rand.Reader
		sk, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
		if err != nil {
			panic(err)
		}

		// Create a new libp2p instance that listen to a random port
		listen, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", p2pPort))
		if err != nil {
			log.Fatalf("Failed to create multiaddr: %s", err.Error())
		}
		host, err := libp2p.New(
			libp2p.ListenAddrs(listen),
			libp2p.Security(noise.ID, noise.New),
			libp2p.Identity(sk),
		)
		if err != nil {
			log.Fatalf("Failed to initialise libp2p instance: %s", err.Error())
		}

		p2pHost = &host
	}
	// Setup mDNS peer discovery
	n := &PeerNotify{
		C: make(chan peer.AddrInfo),
	}
	mdnsSrv := mdns.NewMdnsService(*p2pHost, groupName, n)

	// Create the DHT client using the host
	p2pDHT, err := dht.New(context.Background(), *p2pHost, dht.Mode(dht.ModeAutoServer))
	p2pDHT.Validator = &Validator{}
	if err != nil {
		log.Fatalf("Failed to create Kademlia DHT: %s", err.Error())
	}

	log.Printf("Successfully initialised a host with ID %s", (*p2pHost).ID())

	// Since mdns.NewMdnsService returns an unexported struct, we need to manually export some
	// crucial functions like Start() and Close()
	return &Instance{
		Host:       p2pHost,
		DHT:        p2pDHT,
		GroupName:  groupName,
		PeerNotify: n,
		StartMdns:  mdnsSrv.Start,
		CloseMdns:  mdnsSrv.Close,
	}
}

// Start using mDNS to join this client to the existing cluster
func (i *Instance) Start(ctx context.Context) error {
	// Start trying to connect to new peers
	go i.connect(ctx)
	bc, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Start mDNS for peer discovery
	if err := i.StartMdns(); err != nil {
		return err
	}
	// Connect this DHT client to the DHT cluster
	if err := i.DHT.Bootstrap(bc); err != nil {
		return err
	}
	return nil
}

// Stop shutdown the libp2p host and close this dht client
// It does not destroy the whole DHT itself
func (i *Instance) Stop() error {
	if err := i.DHT.Close(); err != nil {
		return err
	}
	if err := i.CloseMdns(); err != nil {
		return err
	}
	if err := (*i.Host).Close(); err != nil {
		return err
	}
	return nil
}

// connect try to connect to peers discovered by mDNS
func (i *Instance) connect(ctx context.Context) {
	for {
		select {
		case p := <-i.PeerNotify.C:
			err := (*i.Host).Connect(context.Background(), p)
			if err != nil {
				log.Printf("Failed to connect with peer %s: %s", p.ID, err.Error())
				go i.tryConnect(10, p)
				continue
			}
			log.Printf("Successfully establised connection to peer %s", p.ID)
			continue
		case <-ctx.Done():
			return
		}
	}
}

// tryConnect retry connect to the peer discovered
func (i *Instance) tryConnect(cnt int, p peer.AddrInfo) {
	t := time.NewTicker(5 * time.Second)
	for cnt > 0 {
		select {
		case <-t.C:
			err := (*i.Host).Connect(context.Background(), p)
			if err != nil {
				log.Printf("Failed to connect with peer %s: %s", p.ID, err.Error())
				continue
			}
			log.Printf("Successfully establised connection to peer %s", p.ID)
			return
		}
	}
}

// HandlePeerFound will be called when a new peer is found
func (n *PeerNotify) HandlePeerFound(p peer.AddrInfo) {
	n.C <- p
}
