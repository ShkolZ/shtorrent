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

func (msg Message) getLenInt() int {
	return int(binary.BigEndian.Uint16(msg.Length))

}

func MakeMessage(data []byte) (int, *Message, error) {
	if len(data) < 5 {
		return 0, &Message{}, fmt.Errorf("Not enough bytes\n")
	}

	length := data[:4]
	lengthInt := binary.BigEndian.Uint32(length)
	data = data[4:]
	if uint32(len(data)) < lengthInt {
		return 0, &Message{}, fmt.Errorf("Not enought bytes\n")
	}

	other := data[:binary.BigEndian.Uint32(length)]
	if len(other) < 1 {
		return 0, &Message{}, fmt.Errorf("Not Enough bytes\n")
	}
	id := other[0]

	if id == 7 {
		index := other[1:5]
		offset := other[5:9]
		payload := other[9:]
		return int(4 + binary.BigEndian.Uint32(length)), &Message{
			Length:  length,
			Id:      id,
			Index:   &index,
			Offset:  &offset,
			Payload: &payload,
		}, nil
	}

	if len(other) <= 1 {
		return 5, &Message{
			Length: length,
			Id:     id,
		}, nil
	}

	payload := other[1:binary.BigEndian.Uint32(length)]
	return int(4 + binary.BigEndian.Uint32(length)), &Message{
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
