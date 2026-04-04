package main

import (
	"bytes"
	"fmt"
	"log"
	"os"

	"github.com/ShkolZ/shtorrent/config"
	"github.com/ShkolZ/shtorrent/downloader"
	"github.com/ShkolZ/shtorrent/torrent"
)

func main() {
	cfg := &config.Config{}
	if len(os.Args) < 2 {
		log.Fatalln("Not enough arguments")
	}
	torrentPath := os.Args[1]
	fmt.Println(torrentPath)
	data, _ := os.ReadFile(torrentPath)
	br := bytes.NewReader(data)
	bencodef, err := torrent.Open(br)

	if err != nil {
		log.Fatalln(err)
	}

	cfg.Torrent, err = bencodef.BencodeToTorrent()

	// file, err := os.Create(cfg.Torrent.Name)
	// file.Truncate(int64(cfg.Torrent.Length))
	// cfg.File = file
	// log.Fatalln()
	if err != nil {
		log.Fatalln(err)
	}

	downloader.DownloadTorrent(cfg)

}
