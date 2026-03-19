package tracker

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/ShkolZ/shtorrent/config"
	"github.com/jackpal/bencode-go"
)

type TrackerResponse struct {
	Interval      int    `bencode:"interval"`
	TrackerId     string `bencode:"tracker id"`
	Seeders       int    `bencode:"complete"`
	Leechers      int    `bencode:"incomplete"`
	ResponsePeers string `bencode:"peers"`
	MinInterval   int    `bencode:"min interval"`
}

func Announce(cfg *config.Config) (*TrackerResponse, error) {
	peerId := fmt.Sprintf("-ST0001-%v", rand.Text()[:12])
	cfg.Id = peerId
	params := url.Values{}
	params.Set("info_hash", string(cfg.Torrent.InfoHash[:]))
	params.Set("peer_id", peerId)

	queries := fmt.Sprintf("?%v&port=5656&uploaded=0&downloaded=0&left=%v&event=started&compact=1", params.Encode(), cfg.Torrent.Length)
	url := cfg.Torrent.Announce + queries

	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	fmt.Println(string(data))
	tr := TrackerResponse{}
	dataR := bytes.NewReader(data)
	err = bencode.Unmarshal(dataR, &tr)
	if err != nil {
		return nil, err
	}
	return &tr, nil
}
