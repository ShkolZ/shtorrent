package config

import "github.com/ShkolZ/shtorrent/metadata"

type Config struct {
	Id      string
	Torrent *metadata.TorrentFile
}
