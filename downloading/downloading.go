package downloading

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/ShkolZ/shtorrent/config"
	"github.com/jackpal/bencode-go"
)

type Peer struct {
	ip   net.IP
	port uint16
}

type TrackerResponse struct {
	Interval      int     `bencode:"interval"`
	TrackerId     *string `bencode:"tracker id"`
	Seeders       *int    `bencode:"complete"`
	Leechers      *int    `bencode:"incomplete"`
	ResponsePeers string  `bencode:"peers"`
}

func makeConnections(dialer net.Dialer, peers []Peer) chan net.Conn {
	fmt.Println("Connecting to Peers!...")

	connCh := make(chan net.Conn)

	go func() {
		for i := 0; i < len(peers) && i < 25; i++ {

			address := fmt.Sprintf("%v:%v", peers[i].ip, peers[i].port)
			conn, err := dialer.Dial("tcp", address)
			if err == nil {
				connCh <- conn
			} else if err != nil {
				fmt.Printf("Unsuccessful Connection!(%v)", i+1)
			}
		}
	}()
	return connCh
}

func MakePeers(rp string) []Peer {
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

func Announce(cfg *config.Config) (*TrackerResponse, error) {
	peerId := fmt.Sprintf("-ST0001-%v", rand.Text()[:12])
	cfg.Id = peerId
	params := url.Values{}
	params.Set("info_hash", string(cfg.Torrent.InfoHash[:]))
	params.Set("peer_id", peerId)

	queries := fmt.Sprintf("?%v&port=5656&uploaded=0&downloaded=0&left=%v&event=started&compact=1", params.Encode(), cfg.Torrent.Length)
	url := cfg.Torrent.Announce + queries

	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	tr := TrackerResponse{}
	dataR := bytes.NewReader(data)
	err = bencode.Unmarshal(dataR, &tr)
	if err != nil {
		return nil, err
	}
	return &tr, nil
}

// type Handshake struct {
// 	info_hash []byte
// 	peer_id   []byte
// }

func NewHandshake(infoHash []byte, peerId []byte) []byte {

	merged := make([]byte, 0)
	merged = append(merged, byte(19))
	merged = append(merged, []byte("BitTorrent protocol")...)
	merged = append(merged, []byte{0, 0, 0, 0, 0, 0, 0, 0}...)
	merged = append(merged, infoHash...)
	merged = append(merged, peerId...)
	return merged
}

type Message struct {
	Length  []byte
	Id      byte
	Payload *[]byte
}

func (msg Message) getLenInt() int {
	return int(binary.BigEndian.Uint16(msg.Length))

}

func MakeMessage(data []byte) (int, Message) {
	length := data[:4]
	id := data[4]
	payload := data[5 : 5+binary.BigEndian.Uint32(length)-1]
	return int(5 + binary.BigEndian.Uint32(length) - 1), Message{
		Length:  length,
		Id:      id,
		Payload: &payload,
	}
}

func downloadFromPeer(cfg *config.Config, peerCon net.Conn) {
	fmt.Println("Downloading from Peer!...")

	hs := NewHandshake(cfg.Torrent.InfoHash[:], []byte(cfg.Id))
	peerCon.Write(hs)

	handshakeBuff := make([]byte, 68)
	_, err := io.ReadFull(peerCon, handshakeBuff)
	if err != nil && err != io.EOF {
		panic(err)
	}

	buff := make([]byte, 4096)

	n, err := peerCon.Read(buff)
	if err != nil && err != io.EOF {
		panic(err)
	}

	if n == 0 {
		log.Fatalf("Nothing was written into buffer: %v\n", err)
	}
	data := buff[:n]
	n, msg := MakeMessage(data)
	fmt.Printf("Length: %v  ID: %v  Payload: %v\n", msg.Length, msg.Id, msg.Payload)
	data = data[n:]
	n, msg = MakeMessage(data)
	fmt.Printf("Length: %v  ID: %v  Payload: %v\n", msg.Length, msg.Id, msg.Payload)

}

func DownloadTorrent(cfg *config.Config) {
	fmt.Println("Starting to download!...")
	tr, err := Announce(cfg)
	if err != nil {
		panic(err)
	}

	peerSlc := MakePeers(tr.ResponsePeers)

	dialer := net.Dialer{
		Timeout: 5 * time.Second,
	}
	connCh := makeConnections(dialer, peerSlc)

	peerCon := <-connCh

	downloadFromPeer(cfg, peerCon)

	// for i := 0; i < peerAmount; i++ {
	// 	peerCon := <-connCh
	// 	go downloadFromPeer(cfg, peerCon)
	// }

}
