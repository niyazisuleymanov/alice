package main

import (
	"alice/alice"
	"log"
	"os"
)

func main() {
	inputPath := os.Args[1]
	outputPath := os.Args[2]

	torrent := alice.NewTorrent(inputPath, outputPath)

	log.Print("Parsing input")
	_, err := torrent.ParseTorrent()
	if err != nil {
		log.Fatal(err)
	}

	log.Print("Discovering peers")
	torrent.DiscoverPeers()

	log.Print("Starting download")
	torrent.Download()

	log.Print("Creating file(s)")
	torrent.OutputToFile()
}
