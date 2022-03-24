package alice

import (
	"fmt"
	"io"
)

// Handshake string consists of (in order):
//   - 1 byte for pstr length (length of protocal identifier - has to be 19)
//   - 19 bytes for pstr (protocol identifier - BittorentProtocol)
//   - 8 reserved bytes for extension support (no supported here)
//   - 20 bytes for infohash (SHA-1 of bencoded metainfo file)
//   - 20 bytes for peerID (random id to identify ourselves)
type Handshake struct {
	Pstr     string
	InfoHash [20]byte
	PeerID   [20]byte
}

// length of handshake string in bytes
const handshakeLen = 68

// Create new Handshake struct with given infoHash and peerID.
func newHandshake(infoHash, peerID [20]byte) *Handshake {
	return &Handshake{
		Pstr:     "BitTorrent protocol",
		InfoHash: infoHash,
		PeerID:   peerID,
	}
}

// Put together a handshake string.
func (h *Handshake) serializeHandshake() []byte {
	buf := make([]byte, handshakeLen)
	buf[0] = byte(len(h.Pstr)) // len of pstr string in hex
	curr := 1
	curr += copy(buf[curr:], h.Pstr)
	curr += copy(buf[curr:], make([]byte, 8))
	curr += copy(buf[curr:], h.InfoHash[:])
	curr += copy(buf[curr:], h.PeerID[:])
	return buf
}

// Convert raw handshake string into a Handshake struct
func readHandshake(r io.Reader) (*Handshake, error) {
	pstrLenBuf := make([]byte, 1)
	_, err := io.ReadFull(r, pstrLenBuf)
	if err != nil {
		return nil, err
	}
	pstrLen := int(pstrLenBuf[0])
	if pstrLen != 19 {
		err := fmt.Errorf("pstr length should be 19 (0x13) but is %d", pstrLen)
		return nil, err
	}

	handshakeBuf := make([]byte, handshakeLen-1)
	_, err = io.ReadFull(r, handshakeBuf)
	if err != nil {
		return nil, err
	}

	var infoHash, peerID [20]byte
	copy(infoHash[:], handshakeBuf[pstrLen+8:pstrLen+8+20])
	copy(peerID[:], handshakeBuf[pstrLen+8+20:])

	h := Handshake{
		Pstr:     string(handshakeBuf[0:pstrLen]),
		InfoHash: infoHash,
		PeerID:   peerID,
	}
	return &h, nil
}
