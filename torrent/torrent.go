package torrent

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"log"

	"github.com/zeebo/bencode"
)

type FileMetadata struct {
	Length int      `bencode:"length"`
	Path   []string `bencode:"path"`
}
type BencodeInfo struct {
	Length      int            `bencode:"length"`
	Name        string         `bencode:"name"`
	PieceLength int            `bencode:"piece length"`
	Files       []FileMetadata `bencode:"files"`
	Pieces      string         `bencode:"pieces"`
}
type BencodeTorrent struct {
	Announce string
	Info     BencodeInfo `bencode:"info"`
	RawInfo  bencode.RawMessage
}

type RawTorrentInfo struct {
	Announce string             `bencode:"announce"`
	RawInfo  bencode.RawMessage `bencode:"info"`
}

func Open(r io.Reader) (*BencodeTorrent, error) {
	rt := RawTorrentInfo{}
	bt := BencodeTorrent{}

	decoder := bencode.NewDecoder(r)

	err := decoder.Decode(&rt)
	if err != nil {
		return nil, err
	}

	bt.RawInfo = rt.RawInfo
	bt.Announce = rt.Announce

	bencode.DecodeBytes(bt.RawInfo, &bt.Info)

	return &bt, nil
}

type TorrentFile struct {
	Announce    string
	InfoHash    [20]byte
	PieceHashes [][20]byte
	PieceLength int
	Length      int
	Name        string
	Files       []FileMetadata
}

func (bto *BencodeTorrent) BencodeToTorrent() (*TorrentFile, error) {
	fmt.Println("zalupa")

	buff := make([]byte, 0)
	if bto == nil {
		log.Fatalln("aaaaaaaaah")
	}
	buff = bto.RawInfo

	infoHash := sha1.Sum(buff)
	buffer := make([]byte, 100)
	n := hex.Encode(buffer, infoHash[:])
	fmt.Println(string(buffer[:n]))
	fmt.Println(infoHash)

	pieces := []byte(bto.Info.Pieces)
	pieceAmount := len(pieces) / 20

	var piece [20]byte
	pieceHashes := make([][20]byte, pieceAmount)
	for i := 0; i < pieceAmount; i++ {
		for j := 0; j < 20; j++ {
			piece[j] = pieces[i*20+j]
		}
		pieceHashes[i] = piece
	}

	torrentFile := TorrentFile{
		Announce:    bto.Announce,
		InfoHash:    infoHash,
		PieceHashes: pieceHashes,
		PieceLength: bto.Info.PieceLength,
		Length:      bto.Info.Length,
		Name:        bto.Info.Name,
		Files:       bto.Info.Files,
	}

	fmt.Println(torrentFile.PieceLength)

	return &torrentFile, nil
}
