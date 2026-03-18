package main

import (
	"bytes"
	"log"
	"os"

	"github.com/ShkolZ/shtorrent/config"
	"github.com/ShkolZ/shtorrent/downloading"
	"github.com/ShkolZ/shtorrent/metadata"
)

func main() {
	cfg := &config.Config{}
	data, _ := os.ReadFile("/home/ShkolZ/Downloads/S2E10CS.torrent")
	br := bytes.NewReader(data)
	bencodef, err := metadata.Open(br)
	if err != nil {
		log.Fatalln(err)
	}
	cfg.Torrent, err = bencodef.BencodeToTorrent()
	if err != nil {
		log.Fatalln(err)
	}

	downloading.DownloadTorrent(cfg)

}
