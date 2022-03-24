package alice

import (
	"bytes"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/jackpal/bencode-go"
	"github.com/nictuku/dht"
)

// GET request to tracker URL returns:
//   - interval (time to send GET request for list of peers again)
//   - peers (list of peers)
type httpTrackerResponse struct {
	Interval int    `bencode:"interval"`
	Peers    string `bencode:"peers"`
}

func httpRequestPeers(url string) ([]Peer, int, error) {
	// get the response
	conn := &http.Client{Timeout: 5 * time.Second}
	response, err := conn.Get(url)
	if err != nil {
		return nil, 0, err
	}
	defer response.Body.Close()

	// fill body of the response into Peer struct
	trackerResponse := httpTrackerResponse{}
	err = bencode.Unmarshal(response.Body, &trackerResponse)
	if err != nil {
		return nil, 0, err
	}

	peers, err := Unmarshal([]byte(trackerResponse.Peers))
	if err != nil {
		return nil, 0, err
	}
	return peers, trackerResponse.Interval, nil
}

func udpRequestPeers(url string, infoHash, peerID [20]byte, length int) ([]Peer, int, error) {
	raddr, err := net.ResolveUDPAddr("udp", url)
	if err != nil {
		return nil, 0, err
	}
	conn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		return nil, 0, err
	}
	conn.SetDeadline(time.Now().Add(5 * time.Second))
	defer conn.SetDeadline(time.Time{})

	connectReq := newConnect()
	_, err = conn.Write(connectReq.serializeConnect())
	if err != nil {
		return nil, 0, err
	}
	connectBuf := make([]byte, 16)
	_, err = conn.Read(connectBuf)
	if err != nil {
		return nil, 0, err
	}
	connectRes := readConnect(connectBuf)
	if !bytes.Equal(connectReq.TransactionID[:], connectRes.TransactionID[:]) {
		err := fmt.Errorf("expected TID %s received %s", connectReq.TransactionID, connectRes.TransactionID)
		return nil, 0, err
	}
	if connectRes.Action != 0 {
		err := fmt.Errorf("expected action %d (connect) received %d", 0, connectRes.Action)
		return nil, 0, err
	}

	announceReq := newAnnounce(infoHash, peerID, length, connectRes.ConnectionID)
	_, err = conn.Write(announceReq.serializeAnnounce())
	if err != nil {
		return nil, 0, err
	}
	announceBuf := make([]byte, 2048)
	size, err := conn.Read(announceBuf)
	if err != nil {
		return nil, 0, err
	}
	announceRes := readAnnounce(announceBuf[:size])
	if !bytes.Equal(announceReq.TransactionID[:], announceRes.TransactionID[:]) {
		err := fmt.Errorf("expected TID %s received %s", announceReq.TransactionID, announceRes.TransactionID)
		return nil, 0, err
	}
	if announceRes.Action != 1 {
		err := fmt.Errorf("expected action %d (announce) received %d", 1, announceRes.Action)
		return nil, 0, err
	}

	peers, err := Unmarshal([]byte(announceRes.Peers))
	if err != nil {
		return nil, 0, err
	}
	return peers, int(announceRes.Interval), nil
}

func drainResults(n *dht.DHT, peersChannel chan []Peer) {
	for r := range n.PeersRequestResults {
		for _, peers := range r {
			for _, x := range peers {
				peersChannel <- []Peer{toPeer(dht.DecodePeerAddress(x))}
			}
		}
	}
}

// Get list of peers using DHT.
func requestDHTPeers(tf *TorrentFile, peers chan []Peer) error {
	ih := dht.InfoHash(string(tf.InfoHash[:]))
	d, err := dht.New(nil)
	if err != nil {
		return err
	}
	if err = d.Start(); err != nil {
		return err
	}
	go drainResults(d, peers)
	go func() {
		for {
			d.PeersRequest(string(ih), false)
			time.Sleep(5 * time.Second)
		}
	}()
	return nil
}

// Get list of peers from the tracker.
func requestTrackerPeers(tf *TorrentFile, peerID [20]byte, peersChannel chan []Peer) {
	var announceList []string
	if tf.AnnounceList == nil {
		announceList = append(announceList, tf.Announce)
	} else {
		announceList = tf.AnnounceList
	}
	trackerInterval := time.Duration(1)
	go func() {
	Loop:
		ticker := time.NewTicker(trackerInterval * time.Second)
		for {
			select {
			case <-ticker.C:
				for _, announce := range announceList {
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
						peers, interval, err := httpRequestPeers(url)
						if err != nil {
							continue
						}
						peersChannel <- peers
						announceList[0] = announce
						trackerInterval = time.Duration(interval)
						goto Loop
					case "udp":
						peers, interval, err := udpRequestPeers(base.Host, tf.InfoHash, peerID, tf.Length)
						if err != nil {
							continue
						}
						peersChannel <- peers
						announceList[0] = announce
						trackerInterval = time.Duration(interval)
						goto Loop
					}
				}
			}
		}
	}()
}

func (t *Torrent) DiscoverPeers() {
	if t.config.UseTrackers {
		requestTrackerPeers(t.torrentFile, t.peerID, t.peers)
	}
	if t.config.UseDHT {
		requestDHTPeers(t.torrentFile, t.peers)
	}
}
