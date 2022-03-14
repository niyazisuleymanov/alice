package message

import (
	"encoding/binary"
	"fmt"
	"io"
)

type messageID uint8

// Generally every two seconds a message of length zero (keepalives is sent).
//
// All non-keepalive messages with their IDs:
//   - choke 0 (communication channel not ready to receive messages)
//   - unchoke 1 (communication channel ready to receive messages)
//   - interested 2 (communication channel ready to send messages)
//   - not interested 3 (communication channel not ready to send messages)
//   - have 4 (piece index downloader/peer downloaded/has)
//   - bitfield 5 (encode which piece peer is able to send)
//   - request 6 (message payload of the form <index><begin><length> requesting a piece)
//   - piece 7 (message payload of the form <index><begin><block> containing a piece)
//   - cancel 8 (identical to request message used to cancel block requests)
const (
	Choke         messageID = 0
	Unchoke       messageID = 1
	Interested    messageID = 2
	NotInterested messageID = 3
	Have          messageID = 4
	Bitfield      messageID = 5
	Request       messageID = 6
	Piece         messageID = 7
	Cancel        messageID = 8
)

// Every message is of the following form:
// | Message Length | Message ID | Optional Payload |

// Message length is not stored but is just used to parse the message.
type Message struct {
	ID      messageID
	Payload []byte
}

func CreateRequestMessage(index, begin, length int) *Message {
	payload := make([]byte, 12)
	binary.BigEndian.PutUint32(payload[0:4], uint32(index))
	binary.BigEndian.PutUint32(payload[4:8], uint32(begin))
	binary.BigEndian.PutUint32(payload[8:12], uint32(length))
	return &Message{ID: Request, Payload: payload}
}

// Creates peer message with ID of 4 (HAVE).
//
// Format of the message: <length=5><id=4><payload>
func CreateHaveMessage(index int) *Message {
	payload := make([]byte, 4)
	binary.BigEndian.PutUint32(payload, uint32(index))
	return &Message{ID: Have, Payload: payload}
}

// Extract payload (index) from raw HAVE message.
func ReadHaveMessage(msg *Message) (int, error) {
	if msg.ID != Have {
		return -1, fmt.Errorf("expected ID of %d (HAVE), got ID %d", Have, msg.ID)
	}

	if len(msg.Payload) != 4 {
		return -1, fmt.Errorf("expected payload of length 4, got length %d", len(msg.Payload))
	}

	index := int(binary.BigEndian.Uint32(msg.Payload))
	return index, nil
}

// Extract block from raw PIECE message into buf.
func ReadPieceMessage(index int, buf []byte, msg *Message) (int, error) {
	if msg.ID != Piece {
		return 0, fmt.Errorf("expected ID of %d (PIECE), got ID %d", Piece, msg.ID)
	}

	if len(msg.Payload) < 8 {
		return 0, fmt.Errorf("payload too short: %d < 8", len(msg.Payload))
	}

	parsedIndex := int(binary.BigEndian.Uint32(msg.Payload[0:4]))
	if parsedIndex != index {
		return 0, fmt.Errorf("expected index %d, got index %d", index, parsedIndex)
	}

	begin := int(binary.BigEndian.Uint32(msg.Payload[4:8]))
	if begin >= len(buf) {
		return 0, fmt.Errorf("begin offset is larger than payload: %d >= %d", begin, len(buf))
	}

	block := msg.Payload[8:]
	if begin+len(block) > len(buf) {
		return 0, fmt.Errorf("block length [%d] is too long for offset %d with length %d", len(block), begin, len(buf))
	}
	copy(buf[begin:], block)

	return len(block), nil
}

// Put together a message.
func (msg *Message) Serialize() []byte {
	// keepalive
	if msg == nil {
		return make([]byte, 4)
	}

	length := uint32(len(msg.Payload) + 1) // block + ID (1 byte)
	buf := make([]byte, 4+length)
	binary.BigEndian.PutUint32(buf[0:4], length)
	buf[4] = byte(msg.ID)
	copy(buf[5:], msg.Payload)
	return buf
}

// Convert raw message into a Message struct.
func Read(r io.Reader) (*Message, error) {
	bufLen := make([]byte, 4)
	_, err := io.ReadFull(r, bufLen)
	if err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint32(bufLen)

	// keepalive
	if length == 0 {
		return nil, nil
	}

	payloadBuf := make([]byte, length)
	_, err = io.ReadFull(r, payloadBuf)
	if err != nil {
		return nil, err
	}

	msg := Message{
		ID:      messageID(payloadBuf[0]),
		Payload: payloadBuf[1:],
	}

	return &msg, nil
}

func (msg *Message) name() string {
	if msg == nil {
		return "KeepAlive"
	}
	switch msg.ID {
	case Choke:
		return "Choke"
	case Unchoke:
		return "Unchoke"
	case Interested:
		return "Interested"
	case NotInterested:
		return "NotInterested"
	case Have:
		return "Have"
	case Bitfield:
		return "Bitfield"
	case Request:
		return "Request"
	case Piece:
		return "Piece"
	case Cancel:
		return "Cancel"
	default:
		return fmt.Sprintf("unknown message type with ID: %d", msg.ID)
	}
}

func (msg *Message) String() string {
	if msg == nil {
		return msg.name()
	}

	return fmt.Sprintf("%s [%d]", msg.name(), len(msg.Payload))
}
