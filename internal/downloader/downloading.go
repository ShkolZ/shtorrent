package downloader

import (
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"github.com/ShkolZ/shtorrent/internal/config"
	"github.com/ShkolZ/shtorrent/internal/messages"
	"github.com/ShkolZ/shtorrent/internal/peer"
	"github.com/ShkolZ/shtorrent/internal/piece"
	"github.com/ShkolZ/shtorrent/internal/storage"
	"github.com/ShkolZ/shtorrent/internal/tracker"
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

func downloadFromPeer(cfg *config.Config, peerCon *peer.PeerConn, pieceCh chan int, removeCh chan string, pieceDataCh chan *piece.Piece) {
	defer func() {
		fmt.Printf("Peer removed: %v\n", peerCon.Address)
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
			case messages.MsgUnchoke:
				state.Unchoke = true
			case messages.MsgBitfield:
				state.Bitfield = *msg.Payload
			case messages.MsgHave:
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

		for pieceIdx := range pieceCh {
			p, err := piece.GetPiece(peerCon.Conn, pieceIdx, cfg)
			if err != nil {
				fmt.Println(err)
				return
			}
			if piece.CheckHash(p, cfg) {
				pieceDataCh <- p
			} else {
				fmt.Printf("returning piece %v back\n", pieceIdx)
				pieceCh <- pieceIdx
			}

		}

	}
	return

}

func DownloadTorrent(cfg *config.Config) {
	fmt.Println("Starting to download!...")
	tr, err := tracker.Announce(cfg)
	if err != nil {
		panic(err)
	}
	fmt.Println(tr)

	pieceDataCh, err := storage.InitializeFiles(cfg)
	if err != nil {
		log.Fatalln(err)
	}

	pm := peer.NewPeerManager()
	fmt.Println(tr.ResponsePeers)
	go pm.Run(tr.ResponsePeers)

	pieceCh := piece.MakePieceQueue(cfg)

	for peerCon := range pm.OuterConnCh {
		fmt.Println("zalupa")
		go downloadFromPeer(cfg, peerCon, pieceCh, pm.RemoveCh, pieceDataCh)
	}

}
