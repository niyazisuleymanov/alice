package alice

type Torrent struct {
	torrentPath      string
	outputPath       string
	torrentFile      *TorrentFile
	peerID           [20]byte
	trackers         []string
	peers            chan []Peer
	config           Config
	piecesDone       int
	activePeers      int
	finishedDownload bool
	outputBuffer     []byte
}

func NewTorrent(torrentPath, outputPath string) *Torrent {
	return &Torrent{
		torrentPath: torrentPath,
		outputPath:  outputPath,
		peerID:      generatePeerID(),
		peers:       make(chan []Peer),
		config:      DefaultConfig,
		piecesDone:  0,
		activePeers: 0,
	}
}
