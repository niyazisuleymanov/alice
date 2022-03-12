package main

import (
	"alice/file"
	"log"
	"os"
)

func main() {
	inputPath := os.Args[1]
	outputPath := os.Args[2]

	tf, err := file.Open(inputPath)
	if err != nil {
		log.Fatal(err)
	}

	err = tf.DownloadToFile(outputPath)
	if err != nil {
		log.Fatal(err)
	}
}
