package downloading

import (
	"crypto/sha1"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"

	"github.com/ShkolZ/shtorrent/config"
	"github.com/ShkolZ/shtorrent/messages"
	"github.com/ShkolZ/shtorrent/peers"
	"github.com/ShkolZ/shtorrent/pieces"
	"github.com/ShkolZ/shtorrent/tracker"
)

// type Handshake struct {
// 	info_hash []byte
// 	peer_id   []byte
// }

const (
	Choke         = 0
	Unchoke       = 1
	Interested    = 2
	NotInterested = 3
	Have          = 4
	Bitfield      = 5
	Request       = 6
	Piece         = 7
	Cancel        = 8
)

type State struct {
	Unchoke    bool
	Interested bool
	Bitfield   []byte
}

func downloadFromPeer(cfg *config.Config, peerCon *peers.PeerConn, pieceCh chan int, removeCh chan string) {
	defer func() {
		removeCh <- peerCon.Address
	}()
	peerCon.Handshake(cfg)

	state := State{
		Unchoke:    false,
		Interested: false,
	}

	buff := make([]byte, 4096)
	read := 0
	used := 0
	timeout := 0
	for timeout < 3 {
		peerCon.Conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, err := peerCon.Conn.Read(buff[read:])
		if err != nil && err != io.EOF {
			if ne, ok := err.(net.Error); ok {
				if ne.Timeout() {
					timeout++
				} else {
					return
				}
			} else {
				return
			}
		}
		read += n
		used, msg, _ := messages.MakeMessage(buff[used:read])
		used += used
		if msg != nil {
			switch msg.Id {
			case Unchoke:
				state.Unchoke = true
			case Bitfield:
				state.Bitfield = *msg.Payload
			case Have:
				fmt.Println("Not implemented")
				return
			}
		}

	}

	// timeout := 0
	// for timeout < 5 {

	// 	peerCon.Conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	// 	n, err := peerCon.Conn.Read(buff)
	// 	if err == nil && n > 4 {
	// 		for n != 0 {

	// 			data := buff[:n]
	// 			read, msg, err := messages.MakeMessage(data)
	// 			if err != nil {
	// 				fmt.Println(err)
	// 				break
	// 			}

	// 			switch msg.Id {
	// 			case 5:
	// 				state.Bitfield = *msg.Payload
	// 			case 1:
	// 				state.Unchoke = true
	// 			case 0:
	// 				state.Unchoke = false
	// 			case 4:
	// 				return
	// 			}

	// 			buff = buff[read:]
	// 			n -= read

	// 		}

	// 	} else {
	// 		timeout++
	// 	}

	// }

	err := messages.SendInterested(peerCon.Conn)
	if err != nil {
		return
	}
	state.Interested = true

	time.Sleep(3 * time.Second)
	peerCon.Conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, _ := peerCon.Conn.Read(buff)
	_, msg, err := messages.MakeMessage(buff[:n])
	if msg.Id == 1 {
		state.Unchoke = true
	} else {
		return
	}

	buffer := make([]byte, 20048)

	if state.Interested && state.Unchoke {
		pieceDataCh := make(chan pieces.Piece)
		go func() {

			for pieceIdx := range pieceCh {
				currentPiece := make([]byte, cfg.Torrent.PieceLength)
				for offset := 0; offset < (cfg.Torrent.PieceLength/1024)/16; offset++ {
					messages.RequestPiece(peerCon.Conn, pieceIdx, offset)
					peerCon.Conn.SetReadDeadline(time.Now().Add(10 * time.Second))

					block := 16393
					read := 0
					for read <= block {
						n, err := peerCon.Conn.Read(buffer[read:])
						if err != nil && err != io.EOF {
							fmt.Println(err)
							pieceCh <- pieceIdx
							return
						}
						read += n

					}

					data := buffer[:read]
					_, msg, err := messages.MakeMessage(data)
					if err != nil {
						fmt.Println(err)
					}
					if msg.Payload != nil && msg.Id == 7 {
						copy(currentPiece, *msg.Payload)
					}
				}
				pieceHash := sha1.Sum(currentPiece)
				if pieceHash != cfg.Torrent.PieceHashes[pieceIdx] {
					fmt.Println(pieceHash, cfg.Torrent.PieceHashes[pieceIdx])
					fmt.Printf("Returned Piece %d into queue\n", pieceIdx)
					pieceCh <- pieceIdx
				} else {
					pieceDataCh <- pieces.Piece{
						Index: pieceIdx,
						Data:  currentPiece,
					}
				}
			}
		}()

		file, err := os.Create(cfg.Torrent.Name)
		file.Truncate(int64(cfg.Torrent.Length))
		if err != nil {
			log.Fatalln("Couldnt create file")
		}
		for piece := range pieceDataCh {
			writeToFile(file, piece, cfg)
		}

	}

}

func writeToFile(file *os.File, piece pieces.Piece, cfg *config.Config) {
	length := cfg.Torrent.PieceLength
	offset := int64(piece.Index) * int64(length)
	written := 0
	for written < length {
		n, err := file.WriteAt(piece.Data[written:], offset+int64(written))
		if err != nil && err != io.EOF {
			fmt.Println("Some problem with writing file")
		}
		written += n

	}

	fmt.Printf("Wrote Piece %d at offset: %v with length: %v\n", piece.Index, offset/1024, length/1024)

}

func DownloadTorrent(cfg *config.Config) {
	fmt.Println("Starting to download!...")
	tr, err := tracker.Announce(cfg)
	if err != nil {
		panic(err)
	}

	pm := peers.NewPeerManager()
	go pm.Run(tr.ResponsePeers)

	pieceCh := pieces.MakePieceQueue(cfg)

	for peerCon := range pm.OuterConnCh {
		go downloadFromPeer(cfg, peerCon, pieceCh, pm.RemoveCh)
	}

}
