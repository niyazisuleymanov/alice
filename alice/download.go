package alice

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/gosuri/uiprogress"
)

// data is downloaded in blocks (16kB) and not pieces
const maxBlockSize = 16 * 1024

const maxPipelineDepth = 25

type download struct {
	Index  int
	Hash   [20]byte
	Length int
}

type assemble struct {
	Index  int
	Buffer []byte
}

type pieceState struct {
	index         int
	channel       *Channel
	buffer        []byte
	downloaded    int
	requested     int
	pipelineDepth int
}

func (ps *pieceState) readMessage() error {
	msg, err := ps.channel.read()
	if err != nil {
		return err
	}

	// keep-alive
	if msg == nil {
		return nil
	}

	switch msg.ID {
	case unchoke:
		ps.channel.Choked = false
	case choke:
		ps.channel.Choked = true
	case have:
		index, err := readHaveMessage(msg)
		if err != nil {
			return err
		}
		ps.channel.Bitfield.setPiece(index)
	case piece:
		blockLen, err := readPieceMessage(ps.index, ps.buffer, msg)
		if err != nil {
			return err
		}
		ps.downloaded += blockLen
		ps.pipelineDepth--
	}
	return nil
}

func downloadPiece(ch *Channel, d *download) ([]byte, error) {
	state := pieceState{
		index:   d.Index,
		channel: ch,
		buffer:  make([]byte, d.Length),
	}

	ch.Conn.SetDeadline(time.Now().Add(30 * time.Second))
	defer ch.Conn.SetDeadline(time.Time{})

	for state.downloaded < d.Length {
		if !state.channel.Choked {
			// do not exceed maximum pipeline depth and request at most the piece length
			for state.pipelineDepth < maxPipelineDepth && state.requested < d.Length {
				blockSize := maxBlockSize
				// remaining block size might be smaller than 16kB
				if d.Length-state.requested < blockSize {
					blockSize = d.Length - state.requested
				}

				err := ch.sendRequest(d.Index, state.requested, blockSize)
				if err != nil {
					return nil, err
				}
				state.pipelineDepth++
				state.requested += blockSize
			}
		}

		// check status between client and peer
		// might get choked/unchoked/have/piece message
		err := state.readMessage()
		if err != nil {
			return nil, err
		}
	}

	return state.buffer, nil
}

func checkIntegrity(d *download, buf []byte) error {
	hash := sha1.Sum(buf)
	if !bytes.Equal(hash[:], d.Hash[:]) {
		return fmt.Errorf("index %d failed integrity check", d.Index)
	}
	return nil
}

func (t *Torrent) startDownloader(peer Peer, downloadQueue chan *download, assembleQueue chan *assemble) {
	ch, err := t.newChannel(peer, t.peerID, t.torrentFile.InfoHash)
	if err != nil {
		return
	}
	defer ch.Conn.Close()

	ch.sendUnchoke()
	ch.sendInterested()

	for d := range downloadQueue {
		if !ch.Bitfield.hasPiece(d.Index) {
			downloadQueue <- d
			continue
		}

		buf, err := downloadPiece(ch, d)
		if err != nil {
			t.activePeers--
			downloadQueue <- d
			return
		}

		err = checkIntegrity(d, buf)
		if err != nil {
			downloadQueue <- d
			continue
		}

		ch.sendHave(d.Index)
		assembleQueue <- &assemble{d.Index, buf}
	}
}

func calcPieceBounds(tf *TorrentFile, index int) (int, int) {
	begin := index * tf.PieceLength
	end := begin + tf.PieceLength
	if end > tf.Length {
		end = tf.Length
	}
	return begin, end
}

func (t *Torrent) calcPieceSize(index int) int {
	begin, end := calcPieceBounds(t.torrentFile, index)
	return end - begin
}

func (t *Torrent) downloadProgress() *uiprogress.Bar {
	uiprogress.Start()
	bar := uiprogress.AddBar(len(t.torrentFile.PieceHashes))
	bar.AppendCompleted()
	bar.AppendFunc(func(b *uiprogress.Bar) string {
		return "pieces: " + strconv.Itoa(t.piecesDone) + "/" + strconv.Itoa(len(t.torrentFile.PieceHashes))
	})
	bar.AppendFunc(func(b *uiprogress.Bar) string {
		return "peers: " + strconv.Itoa(t.activePeers)
	})
	bar.AppendElapsed()
	return bar
}

func (t *Torrent) assemblePieces(assembleQueue chan *assemble, downloadQueue chan *download) {
	var progressBar *uiprogress.Bar
	if t.config.ShowDownloadProgress {
		progressBar = t.downloadProgress()
	}
	t.outputBuffer = make([]byte, t.torrentFile.Length)
	for t.piecesDone < len(t.torrentFile.PieceHashes) {
		res := <-assembleQueue
		begin, end := calcPieceBounds(t.torrentFile, res.Index)
		copy(t.outputBuffer[begin:end], res.Buffer)
		t.piecesDone++
		if progressBar != nil {
			progressBar.Incr()
		}
	}
	if progressBar != nil {
		uiprogress.Stop()
	}
	close(downloadQueue)
	t.finishedDownload = true
}

func (t *Torrent) Download() {
	downloadQueue := make(chan *download, len(t.torrentFile.PieceHashes))
	assembleQueue := make(chan *assemble)
	for index, hash := range t.torrentFile.PieceHashes {
		length := t.calcPieceSize(index)
		downloadQueue <- &download{Index: index, Hash: hash, Length: length}
	}
	go t.assemblePieces(assembleQueue, downloadQueue)
	for !t.finishedDownload {
		for _, peer := range <-t.peers {
			go t.startDownloader(peer, downloadQueue, assembleQueue)
		}
	}
}

func (t *Torrent) OutputToFile() {
	outFile, err := os.Create(t.outputPath)
	if err != nil {
		log.Fatal(err)
	}
	defer outFile.Close()

	_, err = outFile.Write(t.outputBuffer)
	if err != nil {
		log.Fatal(err)
	}
}
