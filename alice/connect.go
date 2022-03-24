package alice

import (
	"encoding/binary"
)

const connectLen = 16

type Connect struct {
	ProtocolID    uint64 // request & response
	Action        uint32 // request & response
	TransactionID []byte // request & response

	ConnectionID []byte // response
}

func newConnect() *Connect {
	transactionID := generateRandomID(4)
	return &Connect{
		ProtocolID:    0x41727101980,
		Action:        0,
		TransactionID: transactionID,
	}
}

func (c *Connect) serializeConnect() []byte {
	buf := make([]byte, connectLen)
	binary.BigEndian.PutUint64(buf[0:8], c.ProtocolID)
	binary.BigEndian.PutUint32(buf[8:12], c.Action)
	copy(buf[12:16], c.TransactionID[:])
	return buf
}

func readConnect(buf []byte) *Connect {
	connectRequest := make([]byte, connectLen)
	copy(connectRequest, buf)

	actionBuf := make([]byte, 4)
	transactionIDBuf := make([]byte, 4)
	connectionIDBuf := make([]byte, 8)

	copy(actionBuf, connectRequest[0:4])
	copy(transactionIDBuf, connectRequest[4:8])
	copy(connectionIDBuf, connectRequest[8:16])

	cr := Connect{
		Action:        binary.BigEndian.Uint32(actionBuf),
		TransactionID: transactionIDBuf[:],
		ConnectionID:  connectionIDBuf[:],
	}
	return &cr
}
