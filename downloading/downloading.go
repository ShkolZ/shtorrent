package downloading

import (
	"crypto/sha1"
	"fmt"
	"io"
	"log"
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

type State struct {
	Unchoke    bool
	Interested bool
	Bitfield   []byte
}

func downloadFromPeer(cfg *config.Config, peerCon *peers.PeerConn, pieceCh chan int, removeCh chan string) {
	defer func() {
		fmt.Println("Ya tyt duzhe casto")
		removeCh <- peerCon.Address
	}()
	fmt.Println(cfg.Torrent.Length, len(cfg.Torrent.PieceHashes)*cfg.Torrent.PieceLength)
	hs := messages.NewHandshake(cfg.Torrent.InfoHash[:], []byte(cfg.Id))
	peerCon.Conn.Write(hs)
	time.Sleep(2 * time.Second)
	handshakeBuff := make([]byte, 68)
	_, err := io.ReadFull(peerCon.Conn, handshakeBuff)
	if err != nil && err != io.EOF {
		fmt.Println("error getting hasdshake")
		return
	}

	state := State{
		Unchoke:    false,
		Interested: false,
	}

	buff := make([]byte, 4096)
	timeout := 0
	for timeout < 5 {

		peerCon.Conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, err := peerCon.Conn.Read(buff)
		if err == nil && n > 4 {
			for n != 0 {

				data := buff[:n]
				read, msg, err := messages.MakeMessage(data)
				if err != nil {
					fmt.Println(err)
					break
				}

				switch msg.Id {
				case 5:
					state.Bitfield = *msg.Payload
					fmt.Printf("Length: %v  ID: %v  Payload: %v  Read: %v\n", msg.Length, msg.Id, len(*msg.Payload), read)
				case 1:
					state.Unchoke = true
				case 0:
					state.Unchoke = false
				case 4:
					fmt.Println("I am not gonna bother with 'have'")
					peerCon.Conn.Close()
				}

				buff = buff[read:]
				n -= read

			}

		} else {
			timeout++
		}

	}

	err = messages.SendInterested(peerCon.Conn)
	if err != nil {
		return
	}
	state.Interested = true

	fmt.Println("Interested was sent")
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
				currentPiece := make([]byte, 0)
				for offset := 0; offset < (cfg.Torrent.PieceLength/1024)/16; offset++ {
					messages.RequestPiece(peerCon.Conn, pieceIdx, offset)
					peerCon.Conn.SetReadDeadline(time.Now().Add(10 * time.Second))

					block := 16393
					read := 0
					for read <= block {
						n, err := peerCon.Conn.Read(buffer[read:])
						if err != nil && err != io.EOF {
							fmt.Println(err)
							return
						} else if err == io.EOF {
							break
						}
						read += n

					}

					data := buffer[:read]
					_, msg, err := messages.MakeMessage(data)
					if err != nil {
						fmt.Println(err)
					}
					if msg.Payload != nil {
						currentPiece = append(currentPiece, *msg.Payload...)
					}
				}
				pieceHash := sha1.Sum(currentPiece)
				if pieceHash != cfg.Torrent.PieceHashes[pieceIdx] {
					fmt.Printf("Returned Piece %d into queue\n", pieceIdx)
					pieceCh <- pieceIdx
				}
				pieceDataCh <- pieces.Piece{
					Index: pieceIdx,
					Data:  currentPiece,
				}
			}
		}()

		file, err := os.Create(cfg.Torrent.Name)
		if err != nil {
			log.Fatalln("Couldnt create file")
		}
		for piece := range pieceDataCh {
			go writeToFile(file, piece)
		}

	}

}

func writeToFile(file *os.File, piece pieces.Piece) {
	length := len(piece.Data)
	offset := int64(piece.Index) * int64(length)
	written := 0
	for written < length {
		n, err := file.WriteAt(piece.Data, offset)
		if err != nil && err != io.EOF {
			log.Fatalln("Some problem with writing file")
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
