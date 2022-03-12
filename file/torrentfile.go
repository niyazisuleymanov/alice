package file

import (
	"alice/helper"
	"alice/torrent"
	"bytes"
	"crypto/sha1"
	"fmt"
	"os"

	bencode "github.com/jackpal/bencode-go"
)

type TorrentFile struct {
	Announce     string
	AnnounceList []string
	InfoHash     [20]byte
	PieceLength  int
	PieceHashes  [][20]byte
	Length       int
	Name         string
}

type bencodeInfo struct {
	PieceLength int               `bencode:"piece length"`
	Pieces      string            `bencode:"pieces"`
	Length      int               `bencode:"length,omitempty"`
	Name        string            `bencode:"name"`
	Private     bool              `bencode:"private,omitempty"`
	Source      string            `bencode:"source,omitempty"`
	Files       []bencodeFileInfo `bencode:"files,omitempty"`
}

type bencodeTorrent struct {
	Announce     string      `bencode:"announce"`
	AnnounceList [][]string  `bencode:"announce-list"`
	Info         bencodeInfo `bencode:"info"`
}

type bencodeFileInfo struct {
	Length   int      `bencode:"length"`
	Path     []string `bencode:"path"`
	PathUTF8 []string `bencode:"path.utf-8,omitempty"`
}

func (tf *TorrentFile) DownloadToFile(path string) error {
	peerID := helper.GeneratePeerID()
	peers, err := tf.requestPeers(peerID)
	if err != nil {
		return err
	}

	torrent := torrent.Torrent{
		Peers:       peers,
		InfoHash:    tf.InfoHash,
		PieceHashes: tf.PieceHashes,
		PieceLength: tf.PieceLength,
		Length:      tf.Length,
		Name:        tf.Name,
	}

	buf, err := torrent.Download()
	if err != nil {
		return err
	}

	outFile, err := os.Create(path)
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = outFile.Write(buf)
	if err != nil {
		return err
	}
	return nil
}

func Open(path string) (*TorrentFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	bto := bencodeTorrent{}
	err = bencode.Unmarshal(file, &bto)
	if err != nil {
		return nil, err
	}

	return bto.toTorrentFile()
}

func (binfo *bencodeInfo) hash() ([20]byte, error) {
	var buf bytes.Buffer
	err := bencode.Marshal(&buf, *binfo)
	if err != nil {
		return [20]byte{}, err
	}
	h := sha1.Sum(buf.Bytes())
	return h, nil
}

func (binfo *bencodeInfo) generatePieceHashes() ([][20]byte, error) {
	hashLength := 20
	buf := []byte(binfo.Pieces)

	if len(buf)%hashLength != 0 {
		err := fmt.Errorf("received incorrect number of pieces with length %d", len(buf))
		return nil, err
	}

	numHashes := len(buf) / hashLength
	hashes := make([][20]byte, numHashes)

	for i := 0; i < numHashes; i++ {
		copy(hashes[i][:], buf[i*hashLength:(i+1)*hashLength])
	}
	return hashes, nil
}

func (bto *bencodeTorrent) totalLength() (length int) {
	files := bto.Info.Files
	if files != nil {
		for _, f := range files {
			length += f.Length
		}
	} else {
		return bto.Info.Length
	}
	return
}

func flattenAnnounceList(announceList [][]string) []string {
	flat := make([]string, len(announceList))
	for i := 0; i < len(announceList); i++ {
		flat[i] = announceList[i][0]
	}
	return flat
}

func (bto *bencodeTorrent) toTorrentFile() (*TorrentFile, error) {
	infoHash, err := bto.Info.hash()
	if err != nil {
		return nil, err
	}

	pieceHashes, err := bto.Info.generatePieceHashes()
	if err != nil {
		return nil, err
	}

	tf := TorrentFile{
		Announce:     bto.Announce,
		AnnounceList: flattenAnnounceList(bto.AnnounceList),
		InfoHash:     infoHash,
		PieceHashes:  pieceHashes,
		PieceLength:  bto.Info.PieceLength,
		Length:       bto.totalLength(),
		Name:         bto.Info.Name,
	}
	return &tf, nil
}
