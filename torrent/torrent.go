package torrent

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/jackpal/bencode-go"
)

type FileMetadata struct {
	Length int      `bencode:"length"`
	Path   []string `bencode:"path"`
	MdSum  string   `bencode:"md5sum"`
}
type BencodeInfo struct {
	Length      int            `bencode:"length"`
	Name        string         `bencode:"name"`
	PieceLength int            `bencode:"piece length"`
	Files       []FileMetadata `bencode:"files"`
	Pieces      string         `bencode:"pieces"`
}
type BencodeTorrent struct {
	Announce string      `bencode:"announce"`
	Info     BencodeInfo `bencode:"info"`
}

func Open(r io.Reader) (*BencodeTorrent, error) {
	bt := BencodeTorrent{}
	err := bencode.Unmarshal(r, &bt)
	if err != nil {
		return &BencodeTorrent{}, err
	}
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

func (bto BencodeTorrent) BencodeToTorrent() (*TorrentFile, error) {
	var buff bytes.Buffer
	err := bencode.Marshal(&buff, bto.Info)
	if err != nil {
		return &TorrentFile{}, err
	}

	info := buff.Bytes()
	fmt.Println(bto.Info.Files[0].MdSum)
	infoHash := sha1.Sum(info)
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
		// Files:       bto.Info.Files,
	}

	return &torrentFile, nil
}
