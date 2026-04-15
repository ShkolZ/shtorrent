package config

import (
	"os"

	"github.com/ShkolZ/shtorrent/stats"
	"github.com/ShkolZ/shtorrent/torrent"
)

type Config struct {
	Id      string
	Torrent *torrent.TorrentFile
	File    *os.File
	Stats   *stats.Stats
}
