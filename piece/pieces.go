package piece

import (
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/ShkolZ/shtorrent/config"
)

const (
	blockSize int = 16384
)

type Piece struct {
	Index int
	Data  []byte
}

type Block struct {
	Offset int
	Data   []byte
}

func CheckHash(piece *Piece, cfg *config.Config) bool {
	pieceHash := sha1.Sum(piece.Data)
	if pieceHash == cfg.Torrent.PieceHashes[piece.Index] {
		return true
	}
	return false
}

func MakePieceQueue(cfg *config.Config) chan int {
	amount := len(cfg.Torrent.PieceHashes)

	pieceQueueChan := make(chan int, 100)
	go func() {
		for i := range amount {
			pieceQueueChan <- i
		}
	}()

	return pieceQueueChan

}

func GetPiece(peerConn net.Conn, pieceIdx int, cfg *config.Config) (*Piece, error) {
	pieceLen := cfg.Torrent.PieceLength
	pieceData := make([]byte, 0)
	numBlocks := pieceLen / blockSize

	if pieceIdx == len(cfg.Torrent.PieceHashes)-1 {
		pieceLen := cfg.Torrent.Length % pieceLen
		numBlocks = (pieceLen / blockSize) + 1
	}

	for i := range numBlocks {
		offset := i * blockSize

		requestPiece(peerConn, pieceIdx, offset)
		blockBuff := make([]byte, blockSize+13)
		read := 0
		for read < blockSize+13 {
			peerConn.SetReadDeadline(time.Now().Add(10 * time.Second))
			n, err := peerConn.Read(blockBuff[read:])
			if err != nil && err != io.EOF {
				return nil, fmt.Errorf("Unknown error reading blocks: %v\n", err)
			}
			read += n

		}
		blockData := blockBuff[13:]
		pieceData = append(pieceData, blockData...)
	}

	if len(pieceData) != pieceLen {
		return nil, fmt.Errorf("Piece lengths are not equal\n")
	}
	return &Piece{
		Index: pieceIdx,
		Data:  pieceData,
	}, nil
}

func requestPiece(peerConn net.Conn, index int, offset int) {
	buff := make([]byte, 17)

	binary.BigEndian.PutUint32(buff[0:4], 13)
	buff[4] = 6
	binary.BigEndian.PutUint32(buff[5:9], uint32(index))
	binary.BigEndian.PutUint32(buff[9:13], uint32(offset))
	binary.BigEndian.PutUint32(buff[13:17], uint32(16384))

	peerConn.Write(buff)
}
