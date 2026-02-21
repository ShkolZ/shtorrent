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
	fmt.Printf("%v:%v\n", net.IP(bytePeers[:4]), binary.BigEndian.Uint16(bytePeers[4:6]))

}
