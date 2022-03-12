package file

import (
	"alice/announce"
	"alice/connect"
	"alice/peer"
	"bytes"
	"fmt"
	"net"
)

func (tf *TorrentFile) GetPeers(url string, peerID [20]byte) ([]peer.Peer, error) {
	raddr, err := net.ResolveUDPAddr("udp", url)
	if err != nil {
		return nil, err
	}

	conn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	connectReq := connect.New()
	_, err = conn.Write(connectReq.Serialize())
	if err != nil {
		return nil, err
	}

	connectBuf := make([]byte, 2048)
	conn.ReadFromUDP(connectBuf)
	connectRes := connect.Read(connectBuf)

	if !bytes.Equal(connectReq.TransactionID[:], connectRes.TransactionID[:]) {
		err := fmt.Errorf("expected TID %s received %s", connectReq.TransactionID, connectRes.TransactionID)
		return nil, err
	}

	if connectRes.Action != 0 {
		err := fmt.Errorf("expected action %d (connect) received %d", 0, connectRes.Action)
		return nil, err
	}

	announceReq := announce.New(tf.InfoHash, peerID, connectRes.ConnectionID[:], tf.Length)
	_, err = conn.Write(announceReq.Serialize())
	if err != nil {
		return nil, err
	}

	announceBuf := make([]byte, 2048)
	conn.ReadFromUDP(announceBuf)
	announceRes := announce.Read(announceBuf)
	if err != nil {
		return nil, err
	}

	if !bytes.Equal(announceReq.TransactionID[:], announceRes.TransactionID[:]) {
		err := fmt.Errorf("expected TID %s received %s", announceReq.TransactionID, announceRes.TransactionID)
		return nil, err
	}

	if announceRes.Action != 1 {
		err := fmt.Errorf("expected action %d (announce) received %d", 1, announceRes.Action)
		return nil, err
	}

	return peer.Unmarshal([]byte(announceRes.Peers))
}
