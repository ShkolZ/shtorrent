package pieces

import (
	"fmt"

	"github.com/ShkolZ/shtorrent/config"
)

type Piece struct {
	Index int
	Data  []byte
}

func MakePieceQueue(cfg *config.Config) chan int {
	amount := len(cfg.Torrent.PieceHashes)
	fmt.Println(amount)

	pieceQueueChan := make(chan int)
	go func() {
		for i := range amount {
			pieceQueueChan <- i
		}
		close(pieceQueueChan)
	}()

	return pieceQueueChan

}
