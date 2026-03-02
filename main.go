package main

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
	"os"
	"time"

	"github.com/ShkolZ/shtorrent/torrentfile"
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

type Handshake struct {
	info_hash []byte
	peer_id   []byte
}

type Config struct {
	Id string
}

func NewHandshake(ih []byte, pi []byte) Handshake {
	hs := Handshake{
		info_hash: ih,
		peer_id:   pi,
	}
	return hs
}

func (hs Handshake) GetHandshake() []byte {
	merged := make([]byte, 0)
	merged = append(merged, byte(19))
	merged = append(merged, []byte("BitTorrent protocol")...)
	merged = append(merged, []byte{0, 0, 0, 0, 0, 0, 0, 0}...)
	merged = append(merged, hs.info_hash...)
	merged = append(merged, hs.peer_id...)
	return merged
}

func makeConnections(dialer net.Dialer, peers []Peer) chan net.Conn {
	connCh := make(chan net.Conn)
	loop := 0
	go func() {
		for i := 0; i < len(peers) && i < 25; i++ {
			fmt.Println(loop)
			loop++
			address := fmt.Sprintf("%v:%v", peers[i].ip, peers[i].port)

			conn, err := dialer.Dial("tcp", address)
			if err == nil {
				connCh <- conn
			} else if err != nil {
				fmt.Println("zalupa")
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

func Announce(tor *torrentfile.TorrentFile, cfg *Config) (*TrackerResponse, error) {
	peerId := fmt.Sprintf("-ST0001-%v", rand.Text()[:12])
	cfg.Id = peerId
	params := url.Values{}
	params.Set("info_hash", string(tor.InfoHash[:]))
	params.Set("peer_id", peerId)

	log.Println(peerId)
	queries := fmt.Sprintf("?%v&port=5656&uploaded=0&downloaded=0&left=%v&event=started&compact=1", params.Encode(), tor.Length)
	url := tor.Announce + queries
	log.Println(url)
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

func main() {
	cfg := &Config{}
	data, _ := os.ReadFile("/home/ShkolZ/Downloads/debian-13.2.0-amd64-netinst.iso.torrent")
	br := bytes.NewReader(data)
	bencodef, err := torrentfile.Open(br)
	if err != nil {
		log.Fatalln(err)
	}
	torrent, err := bencodef.BencodeToTorrent()
	if err != nil {
		log.Fatalln(err)
	}

	tr, err := Announce(torrent, cfg)
	if err != nil {
		panic(err)
	}

	peerSlc := MakePeers(tr.ResponsePeers)
	fmt.Println(peerSlc)

	dialer := net.Dialer{
		Timeout: 5 * time.Second,
	}
	connCh := makeConnections(dialer, peerSlc)

	conn := <-connCh
	defer conn.Close()

	fmt.Println(conn)

	hs := NewHandshake(torrent.InfoHash[:], []byte(cfg.Id))
	hsMsg := hs.GetHandshake()
	fmt.Println(hsMsg, string(hsMsg))
	conn.Write(hsMsg)
	handshakeBuff := make([]byte, 68)
	_, err = io.ReadFull(conn, handshakeBuff)
	if err != nil && err != io.EOF {
		panic(err)
	}
	buff := make([]byte, 4096)
	n, err := conn.Read(buff)
	if err != nil && err != io.EOF {
		panic(err)
	}
	data = buff[:n]
	message := data[:5]
	payload := data[5:]

	fmt.Printf("Message: %v, Payload: %v", message, payload)
}
