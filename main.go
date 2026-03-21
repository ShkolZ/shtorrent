package main

import (
	"bytes"
	"log"
	"os"

	"github.com/ShkolZ/shtorrent/config"
	"github.com/ShkolZ/shtorrent/downloader"
	"github.com/ShkolZ/shtorrent/torrent"
)

func main() {
	cfg := &config.Config{}
	data, _ := os.ReadFile("/home/ShkolZ/Downloads/S2E10CS.torrent")
	br := bytes.NewReader(data)
	bencodef, err := torrent.Open(br)
	if err != nil {
		log.Fatalln(err)
	}
	cfg.Torrent, err = bencodef.BencodeToTorrent()

	file, err := os.Create(cfg.Torrent.Name)
	file.Truncate(int64(cfg.Torrent.Length))
	cfg.File = file

	if err != nil {
		log.Fatalln(err)
	}

	downloader.DownloadTorrent(cfg)

}
