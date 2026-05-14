package config

import (
	"os"

	"github.com/ShkolZ/shtorrent/internal/metainfo"
	"github.com/ShkolZ/shtorrent/internal/stats"
)

type Config struct {
	Id      string
	Torrent *metainfo.TorrentFile
	File    *os.File
	Stats   *stats.Stats
}
