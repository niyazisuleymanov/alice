package helper

import (
	"math/rand"
	"time"
)

func GeneratePeerID() [20]byte {
	rand.Seed(time.Now().UnixNano())
	symbols := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"
	peerID := [20]byte{}
	for i := 0; i < 20; i++ {
		peerID[i] = symbols[rand.Intn(len(symbols))]
	}
	return peerID
}

func GenerateRandomID(size int) []byte {
	rand.Seed(time.Now().UnixNano())
	symbols := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"
	transactionID := make([]byte, size)
	for i := 0; i < size; i++ {
		transactionID[i] = symbols[rand.Intn(len(symbols))]
	}
	return transactionID
}
