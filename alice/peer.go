package alice

import (
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"strings"
)

type Peer struct {
	IP   net.IP
	Port uint16
}

// Unmarshal peers list from the tracker.
//
// Each peer is 6 bytes long: 4 for IP and 2 for port number.
// Hence, peers list has to be a multiple of 6.
func Unmarshal(peersBinary []byte) ([]Peer, error) {
	// check if peers list is multiple of 6
	// otherwise, return an error
	const peerSize = 6
	if len(peersBinary)%peerSize != 0 {
		err := fmt.Errorf("received malformed binary of peers")
		return nil, err
	}

	// parse peers list into []byte and return
	numPeers := len(peersBinary) / peerSize
	peers := make([]Peer, numPeers)
	for i := 0; i < numPeers; i++ {
		offset := i * peerSize
		peers[i].IP = net.IP(peersBinary[offset : offset+4])
		peers[i].Port = binary.BigEndian.Uint16(peersBinary[offset+4 : offset+6])
	}

	return peers, nil
}

// Return Peer ip and port with suitable format - ip:port
func (p Peer) String() string {
	return net.JoinHostPort(p.IP.String(), strconv.Itoa(int(p.Port)))
}

func toPeer(peer string) Peer {
	addr := strings.Split(peer, ":")
	port, _ := strconv.Atoi(addr[1])
	return Peer{
		IP:   net.ParseIP(addr[0]),
		Port: uint16(port),
	}
}
