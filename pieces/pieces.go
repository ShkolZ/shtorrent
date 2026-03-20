package pieces

import (
	"github.com/ShkolZ/shtorrent/config"
)

type Piece struct {
	Index int
	Data  []byte
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
