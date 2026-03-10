package downloading

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/ShkolZ/shtorrent/config"
	"github.com/jackpal/bencode-go"
)

type Peer struct {
	ip   net.IP
	port uint16
}

type TrackerResponse struct {
	Interval      int     `bencode:"interval"`
	TrackerId     *string `bencode:"tracker id"`
	Seeders       *int    `bencode:"complete"`
	Leechers      *int    `bencode:"incomplete"`
	ResponsePeers string  `bencode:"peers"`
}

func makeConnections(dialer net.Dialer, peers []Peer) chan net.Conn {
	fmt.Println("Connecting to Peers!...")

	connCh := make(chan net.Conn)

	go func() {
		for i := 0; i < len(peers) && i < 25; i++ {

			address := fmt.Sprintf("%v:%v", peers[i].ip, peers[i].port)
			conn, err := dialer.Dial("tcp", address)
			if err == nil {
				connCh <- conn
			} else if err != nil {
				fmt.Printf("Unsuccessful Connection!(%v)\n", i+1)
			}
		}
	}()
	return connCh
}

func makePieceQueue(cfg *config.Config) chan int {
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

func MakePeers(rp string) []Peer {
	bp := []byte(rp)
	peerAmount := len(bp) / 6
	peerSlc := make([]Peer, 0)

	for i := 1; i <= peerAmount; i++ {
		peer := bp[(i-1)*6 : i*6]
		peerSlc = append(peerSlc, Peer{
			ip:   net.IP(peer[:4]),
			port: binary.BigEndian.Uint16(peer[4:6]),
		})
	}

	return peerSlc
}

func Announce(cfg *config.Config) (*TrackerResponse, error) {
	peerId := fmt.Sprintf("-ST0001-%v", rand.Text()[:12])
	cfg.Id = peerId
	params := url.Values{}
	params.Set("info_hash", string(cfg.Torrent.InfoHash[:]))
	params.Set("peer_id", peerId)

	queries := fmt.Sprintf("?%v&port=5656&uploaded=0&downloaded=0&left=%v&event=started&compact=1", params.Encode(), cfg.Torrent.Length)
	url := cfg.Torrent.Announce + queries

	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	tr := TrackerResponse{}
	dataR := bytes.NewReader(data)
	err = bencode.Unmarshal(dataR, &tr)
	if err != nil {
		return nil, err
	}
	return &tr, nil
}

// type Handshake struct {
// 	info_hash []byte
// 	peer_id   []byte
// }

func NewHandshake(infoHash []byte, peerId []byte) []byte {

	merged := make([]byte, 0)
	merged = append(merged, byte(19))
	merged = append(merged, []byte("BitTorrent protocol")...)
	merged = append(merged, []byte{0, 0, 0, 0, 0, 0, 0, 0}...)
	merged = append(merged, infoHash...)
	merged = append(merged, peerId...)
	return merged
}

type Message struct {
	Length  []byte
	Id      byte
	Index   *[]byte
	Offset  *[]byte
	Payload *[]byte
}

func (msg Message) getLenInt() int {
	return int(binary.BigEndian.Uint16(msg.Length))

}

func MakeMessage(data []byte) (int, Message, error) {
	if len(data) < 4 {
		return 0, Message{}, fmt.Errorf("Not enough bytes\n")
	}

	length := data[:4]
	data = data[4:]

	other := data[:binary.BigEndian.Uint32(length)]

	id := other[0]

	if id == 7 {
		index := other[1:5]
		offset := other[5:9]
		payload := other[9:]
		return int(4 + binary.BigEndian.Uint32(length)), Message{
			Length:  length,
			Id:      id,
			Index:   &index,
			Offset:  &offset,
			Payload: &payload,
		}, nil
	}

	if len(other) <= 1 {
		return 5, Message{
			Length: length,
			Id:     id,
		}, nil
	}

	payload := other[1:binary.BigEndian.Uint32(length)]
	return int(4 + binary.BigEndian.Uint32(length)), Message{
		Length:  length,
		Id:      id,
		Payload: &payload,
	}, nil
}

func sendInterested(peerCon net.Conn) error {
	msg := []byte{0, 0, 0, 1, 2}
	_, err := peerCon.Write(msg)
	if err != nil {
		return fmt.Errorf("Some problem sending interested msg(((")
	}
	return nil
}

type State struct {
	Unchoke    bool
	Interested bool
	Bitfield   []byte
}

func requestPiece(peerCon net.Conn, index int, offset int) {
	buff := make([]byte, 17)

	binary.BigEndian.PutUint32(buff[0:4], 13)
	buff[4] = 6
	binary.BigEndian.PutUint32(buff[5:9], uint32(index))
	binary.BigEndian.PutUint32(buff[9:13], uint32(offset*16384))
	binary.BigEndian.PutUint32(buff[13:17], uint32(16384))

	peerCon.Write(buff)
}

func downloadFromPeer(cfg *config.Config, peerCon net.Conn, pieceCh chan int) {
	fmt.Println("Downloading from Peer!...")
	defer peerCon.Close()

	hs := NewHandshake(cfg.Torrent.InfoHash[:], []byte(cfg.Id))
	peerCon.Write(hs)
	time.Sleep(2 * time.Second)
	handshakeBuff := make([]byte, 68)
	_, err := io.ReadFull(peerCon, handshakeBuff)
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

		peerCon.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, err := peerCon.Read(buff)
		if err == nil && n > 4 {
			for n != 0 {

				data := buff[:n]
				read, msg, err := MakeMessage(data)
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
					peerCon.Close()
				}

				buff = buff[read:]
				n -= read

			}

		} else {
			timeout++
		}

	}

	err = sendInterested(peerCon)
	if err != nil {
		return
	}
	state.Interested = true

	fmt.Println("Interested was sent")
	time.Sleep(3 * time.Second)
	peerCon.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, _ := peerCon.Read(buff)
	_, msg, err := MakeMessage(buff[:n])
	if msg.Id == 1 {
		state.Unchoke = true
	} else {
		fmt.Printf("syn shluhi %v\n", err)
	}

	buffer := make([]byte, 20048)
	if state.Interested && state.Unchoke {
		for pieceIdx := range pieceCh {
			for offset := 0; offset < 256/16; offset++ {
				requestPiece(peerCon, pieceIdx, offset)
				peerCon.SetReadDeadline(time.Now().Add(10 * time.Second))

				block := 16393
				read := 0
				for read <= block {
					n, err := peerCon.Read(buffer[read:])
					if err != nil {
						fmt.Println("miron pidor")
						return
					}
					read += n

				}

				data := buffer[:read]
				_, msg, err := MakeMessage(data)
				if err != nil {
					fmt.Println(err)
				}
				fmt.Printf("Id: %v, Index: %v, Offset: %v, Payload: %v\n", msg.Id, *msg.Index, *msg.Offset, len(*msg.Payload))

			}
		}
	}

}

func DownloadTorrent(cfg *config.Config) {
	fmt.Println("Starting to download!...")
	tr, err := Announce(cfg)
	if err != nil {
		panic(err)
	}

	peerSlc := MakePeers(tr.ResponsePeers)

	dialer := net.Dialer{
		Timeout: 5 * time.Second,
	}
	connCh := makeConnections(dialer, peerSlc)
	pieceCh := makePieceQueue(cfg)

	maxPeerAmount := 25

	for i := 0; i < maxPeerAmount; i++ {
		peerCon := <-connCh
		go downloadFromPeer(cfg, peerCon, pieceCh)
	}

}
