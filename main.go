package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/ShkolZ/shtorrent/torrentfile"
)

func main() {
	data, _ := os.ReadFile("/home/shkolz/Downloads/debian-13.2.0-arm64-netinst.iso.torrent")
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
	hexString := hex.EncodeToString(torrent.InfoHash[:])
	log.Println(hexString)
	hexString = strings.ToUpper(hexString)
	anHex := ""
	for i := range hexString {
		anHex += string(hexString[i])
		if i%2 == 1 && i != len(hexString)-1 {
			anHex += "%"
		}
	}
	log.Println(anHex)
	peerId := fmt.Sprintf("-ST0001-%v", rand.Text()[:12])
	log.Println(peerId)
	queries := fmt.Sprintf("?info_hash=%v&peer_id=%v&port=5656&uploaded=0&downloaded=0&left=%v&event=started", anHex, peerId, torrent.Length)
	url := torrent.Announce + queries
	log.Println(url)
	res, err := http.Get(url)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println(res)

}
