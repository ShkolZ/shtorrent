package main

import (
	"bytes"
	"log"
	"os"

	"github.com/ShkolZ/shtorrent/config"
	"github.com/ShkolZ/shtorrent/downloading"
	"github.com/ShkolZ/shtorrent/torrent"
)

func main() {
	cfg := &config.Config{}
	data, _ := os.ReadFile("/home/ShkolZ/Downloads/debian-13.2.0-arm64-netinst.iso.torrent")
	br := bytes.NewReader(data)
	bencodef, err := torrent.Open(br)
	if err != nil {
		log.Fatalln(err)
	}
	cfg.Torrent, err = bencodef.BencodeToTorrent()
	if err != nil {
		log.Fatalln(err)
	}

	downloading.DownloadTorrent(cfg)

}
