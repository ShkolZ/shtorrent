package config

import "github.com/ShkolZ/shtorrent/torrent"

type Config struct {
	Id      string
	Torrent *torrent.TorrentFile
}
