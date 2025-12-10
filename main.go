package main

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"log"
	"os"

	"github.com/ShkolZ/shtorrent/torrentfile"
)

func main() {
	data, _ := os.ReadFile("/home/ShkolZ/Downloads/0F36C10B9452115C303E9C6BA3208560C5D182F8.torrent")
	br := bytes.NewReader(data)
	bencode, err := torrentfile.Open(br)
	if err != nil {
		log.Fatalln(err)
	}
	torrent, err := bencode.BencodeToTorrent()
	if err != nil {
		log.Fatalln(err)
	}
	log.Println(torrent.InfoHash)
	log.Println(torrent.Announce)

	peerId := fmt.Sprintf("-ST0001-%v", rand.Text()[:12])
	log.Println(peerId)
	queries := fmt.Sprintf("?info_hash=%v&peer_id=%v&port=5656&uploaded=0&downloaded=0&left=%v&event=started", peerId, torrent.Length)
	url := torrent.Announce + queries
	log.Println(url)
	// http.Get(url)

}
