package channel

import (
	"alice/bitfield"
	"alice/handshake"
	"alice/message"
	"alice/peer"
	"bytes"
	"fmt"
	"net"
	"time"
)

// Represents the communication channel between client and peer.
type Channel struct {
	Conn     net.Conn          // shared
	Choked   bool              // shared
	Bitfield bitfield.Bitfield // shared
	peer     peer.Peer         // peer data
	infoHash [20]byte          // client data
	peerID   [20]byte          // client data
}

func completeHandshake(conn net.Conn, infoHash, peerID [20]byte) error {
	conn.SetDeadline(time.Now().Add(5 * time.Second))
	defer conn.SetDeadline(time.Time{})

	request := handshake.New(infoHash, peerID) // initialize Handshake struct
	_, err := conn.Write(request.Serialize())  // convert it to connection data
	if err != nil {
		return err
	}

	// convert handshake response to Handshake struct
	result, err := handshake.Read(conn)
	if err != nil {
		return err
	}

	// check if info hash sent equals to the one received
	if !bytes.Equal(result.InfoHash[:], infoHash[:]) {
		err := fmt.Errorf("expected infohash %x but got %x", infoHash, request.InfoHash)
		return err
	}

	return nil
}

// Receive bitfield peer message right after successful handshake.
func receiveBitfield(conn net.Conn) (bitfield.Bitfield, error) {
	conn.SetDeadline(time.Now().Add(5 * time.Second))
	defer conn.SetDeadline(time.Time{})

	msg, err := message.Read(conn)
	if err != nil {
		return nil, err
	}

	if msg == nil {
		err := fmt.Errorf("expected bitfield but got keep alive")
		return nil, err
	}

	if msg.ID != message.Bitfield {
		err := fmt.Errorf("expected bitfield but got message ID %d", msg.ID)
		return nil, err
	}

	return msg.Payload, nil
}

// Create a channel between client and peer.
func New(peer peer.Peer, peerID, infoHash [20]byte) (*Channel, error) {
	conn, err := net.DialTimeout("tcp", peer.String(), 5*time.Second)
	if err != nil {
		return nil, err
	}

	err = completeHandshake(conn, infoHash, peerID)
	if err != nil {
		conn.Close()
		return nil, err
	}

	bf, err := receiveBitfield(conn)
	if err != nil {
		conn.Close()
		return nil, err
	}

	return &Channel{
		Conn:     conn,
		Choked:   true,
		Bitfield: bf,
		peer:     peer,
		infoHash: infoHash,
		peerID:   peerID,
	}, nil
}

func (ch *Channel) Read() (*message.Message, error) {
	msg, err := message.Read(ch.Conn)
	return msg, err
}

func (ch *Channel) SendRequest(index, begin, length int) error {
	req := message.CreateRequestMessage(index, begin, length)
	_, err := ch.Conn.Write(req.Serialize())
	return err
}

func (ch *Channel) SendInterested() error {
	msg := message.Message{ID: message.Interested}
	_, err := ch.Conn.Write(msg.Serialize())
	return err
}

func (ch *Channel) SendNotInterested() error {
	msg := message.Message{ID: message.NotInterested}
	_, err := ch.Conn.Write(msg.Serialize())
	return err
}

func (ch *Channel) SendUnchoke() error {
	msg := message.Message{ID: message.Unchoke}
	_, err := ch.Conn.Write(msg.Serialize())
	return err
}

func (ch *Channel) SendHave(index int) error {
	msg := message.CreateHaveMessage(index)
	_, err := ch.Conn.Write(msg.Serialize())
	return err
}
