package file

import (
	"alice/announce"
	"alice/connect"
	"alice/peer"
	"bytes"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/jackpal/bencode-go"
)

// GET request to tracker URL returns:
//   - interval (time to send GET request for list of peers again)
//   - peers (list of peers)
type httpTrackerResponse struct {
	Interval int    `bencode:"interval"`
	Peers    string `bencode:"peers"`
}

func httpRequestPeers(url string) ([]peer.Peer, error) {
	// get the response
	conn := &http.Client{Timeout: 5 * time.Second}
	response, err := conn.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	// fill body of the response into Peer struct
	trackerResponse := httpTrackerResponse{}
	err = bencode.Unmarshal(response.Body, &trackerResponse)
	if err != nil {
		return nil, err
	}

	return peer.Unmarshal([]byte(trackerResponse.Peers))
}

func udpRequestPeers(url string, infoHash, peerID [20]byte, length int) ([]peer.Peer, error) {
	raddr, err := net.ResolveUDPAddr("udp", url)
	if err != nil {
		return nil, err
	}

	conn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		return nil, err
	}
	conn.SetDeadline(time.Now().Add(5 * time.Second))
	defer conn.SetDeadline(time.Time{})

	connectReq := connect.New()
	_, err = conn.Write(connectReq.Serialize())
	if err != nil {
		return nil, err
	}

	connectBuf := make([]byte, 16)
	_, err = conn.Read(connectBuf)
	if err != nil {
		return nil, err
	}
	connectRes := connect.Read(connectBuf)

	if !bytes.Equal(connectReq.TransactionID[:], connectRes.TransactionID[:]) {
		err := fmt.Errorf("expected TID %s received %s", connectReq.TransactionID, connectRes.TransactionID)
		return nil, err
	}

	if connectRes.Action != 0 {
		err := fmt.Errorf("expected action %d (connect) received %d", 0, connectRes.Action)
		return nil, err
	}

	announceReq := announce.New(infoHash, peerID, length, connectRes.ConnectionID)
	_, err = conn.Write(announceReq.Serialize())
	if err != nil {
		return nil, err
	}

	announceBuf := make([]byte, 2048)
	size, err := conn.Read(announceBuf)
	if err != nil {
		return nil, err
	}
	announceRes := announce.Read(announceBuf[:size])

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

// Get list of peers from the announcer.
func (tf *TorrentFile) requestPeers(peerID [20]byte) ([]peer.Peer, error) {
	var announceList []string
	if tf.AnnounceList == nil {
		announceList = append(announceList, tf.Announce)
	} else {
		announceList = tf.AnnounceList
	}
	for _, announce := range announceList {
		log.Printf("Try connecting to: %s\n", announce)
		base, err := url.Parse(announce)
		if err != nil {
			continue
		}
		switch base.Scheme {
		case "http", "https":
			params := url.Values{
				"info_hash":  []string{string(tf.InfoHash[:])},
				"peer_id":    []string{string(peerID[:])},
				"port":       []string{strconv.Itoa(0)},
				"uploaded":   []string{"0"},
				"downloaded": []string{"0"},
				"compact":    []string{"1"},
				"left":       []string{strconv.Itoa(tf.Length)},
			}
			base.RawQuery = params.Encode()
			url := base.String()
			peers, err := httpRequestPeers(url)
			if err != nil {
				continue
			}
			return peers, nil
		case "udp":
			peers, err := udpRequestPeers(base.Host, tf.InfoHash, peerID, tf.Length)
			if err != nil {
				continue
			}
			return peers, nil
		}
	}
	err := fmt.Errorf("no tracker was reached")
	return nil, err
}
