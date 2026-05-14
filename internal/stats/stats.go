package stats

import (
	"fmt"
	"sync/atomic"
	"time"
)

type Stats struct {
	Downloaded uint64
	Uploaded   uint64
	PiecesDone uint64
}

func StartLogging(stats *Stats, pieceAm int) {
	ticker := time.NewTicker(time.Second)
	go func() {
		for {
			<-ticker.C
			downloaded := atomic.LoadUint64(&stats.Downloaded)
			pieces := atomic.LoadUint64(&stats.PiecesDone)
			fmt.Printf("Downloaded: %v bytes Pieces: %v/%v\n", downloaded, pieces, pieceAm)

		}
	}()
}

func OnDownload(i int) {

}

func OnPieceDone() {

}
