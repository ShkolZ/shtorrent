package downloader

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"

	"github.com/ShkolZ/shtorrent/config"
	"github.com/ShkolZ/shtorrent/messages"
	"github.com/ShkolZ/shtorrent/peer"
	"github.com/ShkolZ/shtorrent/piece"
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

func downloadFromPeer(cfg *config.Config, peerCon *peer.PeerConn, pieceCh chan int, removeCh chan string) {
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
		peerCon.Conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		n, err := peerCon.Conn.Read(buff[read:])
		if err != nil && err != io.EOF {
			fmt.Println(err)
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

	if state.Interested && state.Unchoke {
		pieceDataCh := make(chan *piece.Piece)
		go func() {
			for pieceIdx := range pieceCh {
				p, err := piece.GetPiece(peerCon.Conn, pieceIdx, cfg)
				if err != nil {
					fmt.Println(err)
					pieceCh <- pieceIdx
					close(pieceDataCh)
					return
				}
				if piece.CheckHash(p, cfg) {
					pieceDataCh <- p
				} else {
					fmt.Println("returning piece %v back\n", pieceIdx)
					pieceCh <- pieceIdx
				}

			}
		}()

		if err != nil {
			log.Fatalln("Couldnt create file")
		}
		for piece := range pieceDataCh {
			writeToFile(cfg.File, piece, cfg)
		}

	}
	return

}

func writeToFile(file *os.File, p *piece.Piece, cfg *config.Config) {
	length := cfg.Torrent.PieceLength
	offset := int64(p.Index) * int64(length)
	written := 0
	for written < length {
		n, err := file.WriteAt(p.Data[written:], offset+int64(written))
		if err != nil && err != io.EOF {
			fmt.Println("Some problem with writing file")
		}
		written += n

	}

	fmt.Printf("Wrote Piece %d at offset: %v with length: %v\n", p.Index, offset/1024, length/1024)

}

func DownloadTorrent(cfg *config.Config) {
	fmt.Println("Starting to download!...")
	tr, err := tracker.Announce(cfg)
	if err != nil {
		panic(err)
	}

	pm := peer.NewPeerManager()
	go pm.Run(tr.ResponsePeers)

	pieceCh := piece.MakePieceQueue(cfg)

	for peerCon := range pm.OuterConnCh {
		go downloadFromPeer(cfg, peerCon, pieceCh, pm.RemoveCh)
	}

}
