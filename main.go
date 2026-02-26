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

	"github.com/ShkolZ/shtorrent/torrentfile"
	"github.com/jackpal/bencode-go"
)

type Peer struct {
	ip   *string `bencode:"ip"`
	port *string `bencode:"port"`
}

type TrackerResponse struct {
	Interval  int     `bencode:"interval"`
	TrackerId *string `bencode:"tracker id"`
	Seeders   *int    `bencode:"complete"`
	Leechers  *int    `bencode:"incomplete"`
	Peers     string  `bencode:"peers"`
}

type Handshake struct {
	info_hash []byte
	peer_id   []byte
}

func NewHandshake(ih []byte, pi []byte) Handshake {
	hs := Handshake{
		info_hash: ih,
		peer_id:   pi,
	}
	return hs
}

func (hs Handshake) GetHandshake() {
	merged := make([]byte, 0)
	merged = append(merged, byte(19))
	merged = append(merged, []byte("BitTorrent protocol")...)
	merged = append(merged, []byte{0, 0, 0, 0, 0, 0, 0, 0}...)
	merged = append(merged, hs.info_hash...)
	merged = append(merged, hs.peer_id...)

}

func main() {
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

	log.Println(torrent.InfoHash)
	log.Println(torrent.Announce)

	peerId := fmt.Sprintf("-ST0001-%v", rand.Text()[:12])

	params := url.Values{}
	params.Set("info_hash", string(torrent.InfoHash[:]))
	params.Set("peer_id", peerId)

	log.Println(peerId)
	queries := fmt.Sprintf("?%v&port=5656&uploaded=0&downloaded=0&left=%v&event=started&compact=1", params.Encode(), torrent.Length)
	url := torrent.Announce + queries
	log.Println(url)
	res, err := http.Get(url)
	if err != nil {
		log.Fatalln(err)
	}
	defer res.Body.Close()

	data, err = io.ReadAll(res.Body)
	if err != nil {
		log.Fatalln(err)
	}

	tr := TrackerResponse{}
	dataR := bytes.NewReader(data)
	err = bencode.Unmarshal(dataR, &tr)
	if err != nil {
		log.Fatalln(err)
	}
	bytePeers := []byte(tr.Peers)
	address := fmt.Sprintf("%v:%v", net.IP(bytePeers[:4]), binary.BigEndian.Uint16(bytePeers[4:6]))

	addr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println(addr)
	connection, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		log.Fatalln(err)
	}
	defer connection.Close()

	fmt.Println(connection)
	connection.Write([]byte("zalupa"))
	buff := make([]byte, 4096)
	var pointer int
	for {
		n, err := connection.Read(buff[pointer:])
		if err != nil {
			log.Fatalln(err)
		}
		if n == 0 {
			break
		}
		pointer += n
		fmt.Println(pointer, buff)

	}
	fmt.Println(string(buff))
}
