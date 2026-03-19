package messages

import (
	"encoding/binary"
	"fmt"
	"net"
)

type Message struct {
	Length  []byte
	Id      byte
	Index   *[]byte
	Offset  *[]byte
	Payload *[]byte
}

func RequestPiece(peerCon net.Conn, index int, offset int) {
	buff := make([]byte, 17)

	binary.BigEndian.PutUint32(buff[0:4], 13)
	buff[4] = 6
	binary.BigEndian.PutUint32(buff[5:9], uint32(index))
	binary.BigEndian.PutUint32(buff[9:13], uint32(offset*16384))
	binary.BigEndian.PutUint32(buff[13:17], uint32(16384))

	peerCon.Write(buff)
}

func NewHandshake(infoHash []byte, peerId []byte) []byte {

	merged := make([]byte, 0)
	merged = append(merged, byte(19))
	merged = append(merged, []byte("BitTorrent protocol")...)
	merged = append(merged, []byte{0, 0, 0, 0, 0, 0, 0, 0}...)
	merged = append(merged, infoHash...)
	merged = append(merged, peerId...)
	return merged

}

func (msg Message) getLenInt() int {
	return int(binary.BigEndian.Uint16(msg.Length))

}

func MakeMessage(data []byte) (int, Message, error) {
	if len(data) < 5 {
		return 0, Message{}, fmt.Errorf("Not enough bytes\n")
	}

	length := data[:4]
	data = data[4:]

	other := data[:binary.BigEndian.Uint32(length)]
	if len(other) < 1 {
		return 0, Message{}, fmt.Errorf("Not Enough bytes\n")
	}
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

func SendInterested(peerCon net.Conn) error {
	msg := []byte{0, 0, 0, 1, 2}
	_, err := peerCon.Write(msg)
	if err != nil {
		return fmt.Errorf("Some problem sending interested msg(((")
	}
	return nil
}
