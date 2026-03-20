package peers

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/ShkolZ/shtorrent/config"
)

const target int = 25

type Peer struct {
	ip   net.IP
	port uint16
}

type PeerConn struct {
	Address string
	Conn    net.Conn
}

func (pc *PeerConn) Handshake(cfg *config.Config) (bool, error) {
	hsMsg := newHandshake(cfg)
	if _, err := pc.Conn.Write(hsMsg); err != nil {
		return false, fmt.Errorf("Error with sending handshake: %v\n", err)
	}

	timeout := 3
	read := 0
	buffer := make([]byte, 68)
	for read < len(hsMsg) {
		pc.Conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, err := pc.Conn.Read(buffer[read:])
		if err != nil && err != io.EOF {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				timeout++
			}
		} else {
			return false, fmt.Errorf("Problem reading handshake: %v\n", err)
		}
		read += n
	}

	return true, nil

}

func newHandshake(cfg *config.Config) []byte {

	merged := make([]byte, 0)
	merged = append(merged, byte(19))
	merged = append(merged, []byte("BitTorrent protocol")...)
	merged = append(merged, []byte{0, 0, 0, 0, 0, 0, 0, 0}...)
	merged = append(merged, cfg.Torrent.InfoHash[:]...)
	merged = append(merged, cfg.Id...)
	return merged

}

type PeerManager struct {
	peerMap map[string]*PeerConn
	mutex   sync.Mutex

	connCh      chan *PeerConn
	OuterConnCh chan *PeerConn
	RemoveCh    chan string
}

func NewPeerManager() *PeerManager {
	return &PeerManager{
		peerMap:     make(map[string]*PeerConn),
		connCh:      make(chan *PeerConn),
		OuterConnCh: make(chan *PeerConn),
		RemoveCh:    make(chan string),
	}
}

func (pm *PeerManager) Run(rp string) {
	peers := makePeers(rp)
	go pm.fillConnections(peers)

	for {
		select {
		case addr := <-pm.RemoveCh:
			pm.mutex.Lock()
			peerCon, ok := pm.peerMap[addr]
			if ok {
				peerCon.Conn.Close()
				delete(pm.peerMap, addr)
			}
			pm.mutex.Unlock()
		case conn := <-pm.connCh:

			pm.mutex.Lock()
			pm.peerMap[conn.Address] = conn
			pm.OuterConnCh <- conn
			pm.mutex.Unlock()
		}
	}
}

func (pm *PeerManager) fillConnections(peers []Peer) {
	ticker := time.NewTicker(1 * time.Second)
	i := 0
	for {
		<-ticker.C

		pm.mutex.Lock()
		pieceAmount := len(pm.peerMap)
		pm.mutex.Unlock()
		if pieceAmount < target && i < len(peers) {
			conn, err := makeConnection(peers[i])
			if err == nil {
				pm.connCh <- conn
			}

		} else if i > len(peers) {
			ticker.Stop()
			break
		}

		i++
	}

}

func makePeers(rp string) []Peer {
	bp := []byte(rp)
	peerAmount := len(bp) / 6
	peerSlc := make([]Peer, 0)

	for i := 1; i <= peerAmount; i++ {
		peer := bp[(i-1)*6 : i*6]
		peerSlc = append(peerSlc, Peer{
			ip:   net.IP(peer[:4]),
			port: binary.BigEndian.Uint16(peer[4:6]),
		})
	}

	return peerSlc
}

func makeConnection(peer Peer) (*PeerConn, error) {

	address := fmt.Sprintf("%v:%v", peer.ip, peer.port)
	conn, err := net.DialTimeout("tcp", address, 3*time.Second)

	if err != nil {
		return nil, err
	}
	return &PeerConn{
		Address: address,
		Conn:    conn,
	}, nil
}

func MakeConnections(dialer net.Dialer, peers []Peer) chan net.Conn {
	fmt.Println("Connecting to Peers!...")

	connCh := make(chan net.Conn)

	go func() {
		for i := 0; i < len(peers) && i < 25; i++ {

			address := fmt.Sprintf("%v:%v", peers[i].ip, peers[i].port)
			conn, err := dialer.Dial("tcp", address)
			if err == nil {
				connCh <- conn
			} else if err != nil {
				fmt.Printf("Unsuccessful Connection!(%v)\n", i+1)
			}
		}
	}()
	return connCh
}
