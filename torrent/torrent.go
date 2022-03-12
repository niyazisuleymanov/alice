package torrent

import (
	"alice/channel"
	"alice/message"
	"alice/peer"
	"bytes"
	"crypto/sha1"
	"fmt"
	"log"
	"time"
)

// data is downloaded in blocks (16kB) and not pieces
const MaxBlockSize = 16 * 1024

const MaxPipelineDepth = 5

type Torrent struct {
	Peers       []peer.Peer
	PeerID      [20]byte
	InfoHash    [20]byte
	PieceHashes [][20]byte
	PieceLength int
	Length      int
	Name        string
}

type download struct {
	index  int
	hash   [20]byte
	length int
}

type assemble struct {
	index  int
	buffer []byte
}

type pieceState struct {
	index         int
	channel       *channel.Channel
	buffer        []byte
	downloaded    int
	requested     int
	pipelineDepth int
}

func (ps *pieceState) readMessage() error {
	msg, err := ps.channel.Read()
	if err != nil {
		return err
	}

	// keep-alive
	if msg == nil {
		return nil
	}

	switch msg.ID {
	case message.Unchoke:
		ps.channel.Choked = false
	case message.Choke:
		ps.channel.Choked = true
	case message.Have:
		index, err := message.ReadHaveMessage(msg)
		if err != nil {
			return err
		}
		ps.channel.Bitfield.SetPiece(index)
	case message.Piece:
		blockLen, err := message.ReadPieceMessage(ps.index, ps.buffer, msg)
		if err != nil {
			return err
		}
		ps.downloaded += blockLen
		ps.pipelineDepth--
	}
	return nil
}

func downloadPiece(ch *channel.Channel, d *download) ([]byte, error) {
	state := pieceState{
		index:   d.index,
		channel: ch,
		buffer:  make([]byte, d.length),
	}

	ch.Conn.SetDeadline(time.Now().Add(30 * time.Second))
	defer ch.Conn.SetDeadline(time.Time{})

	for state.downloaded < d.length {
		if !state.channel.Choked {
			// do not exceed maximum pipeline depth and request at most the piece length
			for state.pipelineDepth < MaxPipelineDepth && state.requested < d.length {
				blockSize := MaxBlockSize
				// remaining block size might be smaller than 16kB
				if d.length-state.requested < blockSize {
					blockSize = d.length - state.requested
				}

				err := ch.SendRequest(d.index, state.requested, blockSize)
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
	if !bytes.Equal(hash[:], d.hash[:]) {
		return fmt.Errorf("index %d failed integrity check", d.index)
	}
	return nil
}

func (t *Torrent) startDownloader(peer peer.Peer, downloadQueue chan *download, assembleQueue chan *assemble) {
	ch, err := channel.New(peer, t.PeerID, t.InfoHash)
	if err != nil {
		log.Printf("Could not handshake with %s. Disconnecting...\n", peer.IP)
		return
	}
	defer ch.Conn.Close()
	log.Printf("Completed handshake with %s\n", peer.IP)

	ch.SendUnchoke()
	ch.SendInterested()

	for d := range downloadQueue {
		if !ch.Bitfield.HasPiece(d.index) {
			downloadQueue <- d
			continue
		}

		buf, err := downloadPiece(ch, d)
		if err != nil {
			log.Println("Exiting", err)
			downloadQueue <- d // did not download the piece, try again
			return
		}

		err = checkIntegrity(d, buf)
		if err != nil {
			log.Printf("Piece #%d failed integrity check\n", d.index)
			downloadQueue <- d // did not download the piece, try again
			continue
		}

		ch.SendHave(d.index)
		assembleQueue <- &assemble{d.index, buf}
	}
}

func (t *Torrent) calcPieceBounds(index int) (int, int) {
	begin := index * t.PieceLength
	end := begin + t.PieceLength
	if end > t.Length {
		end = t.Length
	}
	return begin, end
}

func (t *Torrent) calcPieceSize(index int) int {
	begin, end := t.calcPieceBounds(index)
	return end - begin
}

func (t *Torrent) Download() ([]byte, error) {
	downloadQueue := make(chan *download, len(t.PieceHashes))
	assembleQueue := make(chan *assemble)

	for index, hash := range t.PieceHashes {
		length := t.calcPieceSize(index)
		downloadQueue <- &download{index, hash, length}
	}

	for _, peer := range t.Peers {
		go t.startDownloader(peer, downloadQueue, assembleQueue)
	}

	buffer := make([]byte, t.Length)
	donePieces := 0
	for donePieces < len(t.PieceHashes) {
		res := <-assembleQueue
		begin, end := t.calcPieceBounds(res.index)
		copy(buffer[begin:end], res.buffer)
		donePieces++

		percent := float64(donePieces) / float64(len(t.PieceHashes)) * 100
		log.Printf("(%0.2f%%) Downloaded piece #%d\n", percent, res.index)
	}

	close(downloadQueue)

	return buffer, nil
}
