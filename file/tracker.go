package file

import (
	"alice/peer"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/jackpal/bencode-go"
)

// GET request to tracker URL returns:
//   - interval (time to send GET request for list of peers again)
//   - peers (list of peers)
type bencodeTrackerResponse struct {
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
	trackerResponse := bencodeTrackerResponse{}
	err = bencode.Unmarshal(response.Body, &trackerResponse)
	if err != nil {
		return nil, err
	}

	return peer.Unmarshal([]byte(trackerResponse.Peers))
}

// Get list of peers from the announcer.
func (tf *TorrentFile) requestPeers(peerID [20]byte) ([]peer.Peer, error) {
	base, err := url.Parse(tf.Announce)
	if err != nil {
		return nil, err
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
		return httpRequestPeers(url)
	case "udp":
		peers, err := tf.GetPeers(base.Host, peerID)
		if err != nil {
			return nil, err
		}

		return peers, nil
	default:
		err := fmt.Errorf("bad or unsupported url scheme")
		return nil, err
	}
}
