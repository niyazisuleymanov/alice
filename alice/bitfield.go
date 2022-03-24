package alice

// Is only sent as the first message immediately after handshake.
// Used to efficiently encode which pieces peers are able to send.
// Note: pieces are zero indexed
//
// Example:
//   - [0 0 1 0 1 0 0 0] (only pieces 2 and 4 are available)
//   - [1 1 1 1 1 1 1 1] (only pieces in the interval [0, 7] are available)
//   - [0 0 0 0 0 0 0 0] [0 0 0 0 0 0 0 1] (only piece 15 is available)
type Bitfield []byte

// Check if piece at the given index can be sent by peer(s).
func (bf Bitfield) hasPiece(index int) bool {
	bfIndex := index / 8 // determine which bitfield we need
	offset := index % 8  // determine offset within that bitfield

	return bf[bfIndex]>>(7-offset)&1 != 0
}

// Set piece at the given index as available to be sent by peer(s).
func (bf Bitfield) setPiece(index int) {
	byteIndex := index / 8
	offset := index % 8

	bf[byteIndex] |= 1 << (7 - offset)
}
