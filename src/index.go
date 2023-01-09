/*
 * Authors: Leo Vainio, lvainio@kth.se &
 */

package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/bjarneh/latinx"
)

var charValues map[byte]int64

// init Creates a map with each letters value.
func init() {
	converter := latinx.Get(latinx.Windows1252) // extended latin-1
	alphabet, _, err := converter.Encode([]byte(" abcdefghijklmnopqrstuvwxyzäåö"))
	checkErr(err)

	// Give characters a, b, ..., ö values 1, 2, ..., 29
	charValues = make(map[byte]int64)
	for k, c := range alphabet {
		charValues[c] = int64(k)
	}
}

// main searches rawindex.txt and saves all pointers to the first occurrence of each 3-letter combination
// and stores the pointers in a file called index.
func main() {
	start := time.Now()

	rawIndex, err := os.Open("/afs/kth.se/misc/info/kurser/DD2350/adk22/labb1/rawindex.txt")
	checkErr(err)
	s, err := rawIndex.Stat()
	checkErr(err)
	rawIndexSize := s.Size()

	index, err := os.Create("index")
	checkErr(err)

	eof := false
	prevHash := int64(-1)
	currHash := int64(0)
	var ptr int64

	for !eof {
		for {
			// Get current position
			ptr, err = rawIndex.Seek(0, io.SeekCurrent)
			checkErr(err)

			// Calculate hash of the new word
			var c1, c2, c3 byte
			c1, c2, c3, eof = readThreeBytes(rawIndex)
			if eof {
				eof = true
				break
			}
			currHash = hash(c1, c2, c3)

			// Read bytes until a newline is encountered
			eof = seekNewLine(rawIndex)
			if eof {
				eof = true
				break
			}

			// We found the next hash
			if currHash != prevHash {
				diff := currHash - prevHash
				for i := 0; int64(i) < diff; i++ {
					err = binary.Write(index, binary.LittleEndian, ptr)
					fmt.Println(ptr, currHash)
					checkErr(err)
				}

				break
			}
		}
		prevHash = currHash
		if ptr < rawIndexSize-5000 {
			nextHash(currHash, rawIndex) // Seek forward some bytes.
		}
	}

	s, err = index.Stat()
	checkErr(err)
	numBytes := s.Size()
	duration := time.Since(start)
	fmt.Println("Index file completed, construction took:", duration, "Total bytes written:", numBytes)

	rawIndex.Close()
	index.Close()
}

// nextHash approximately finds the position of the next hash by jumping forward in the file.
func nextHash(prevHash int64, rawIndex *os.File) {
	prevPtr, err := rawIndex.Seek(0, io.SeekCurrent)
	checkErr(err)

	const dist int64 = 2500
	var nextPtr int64
	for {
		// Seek <dist> bytes forward in file.
		_, err = rawIndex.Seek(dist, io.SeekCurrent)
		checkErr(err)

		// Go to a new line.
		seekNewLine(rawIndex)
		nextPtr, err = rawIndex.Seek(0, io.SeekCurrent)
		checkErr(err)

		// Hash word at the new line.
		c1, c2, c3, _ := readThreeBytes(rawIndex)
		h := hash(c1, c2, c3)

		// New hash -> go back to prevPtr in order to find the first occurence of this hash.
		if h != prevHash {
			_, err = rawIndex.Seek(prevPtr, io.SeekStart)
			checkErr(err)
			return
		}
		prevPtr = nextPtr
	}
}

// readThreeBytes reads three bytes from rawindex.txt and returns them. Returns true if eof.
func readThreeBytes(rawIndex *os.File) (byte, byte, byte, bool) {
	var b []byte = make([]byte, 3)
	_, err := rawIndex.Read(b)
	if checkEOF(err) {
		return 0, 0, 0, true
	}
	return b[0], b[1], b[2], false
}

// seekNewLine reads bytes until a new line is read. Returns true if eof.
func seekNewLine(rawIndex *os.File) bool {
	var b []byte = make([]byte, 1)
	for b[0] != '\n' {
		_, err := rawIndex.Read(b)
		if checkEOF(err) {
			return true
		}
	}
	return false
}

// hash hashes the first three letters of a word, each possible three letter combination returns a unique hash.
func hash(c1, c2, c3 byte) (hash int64) {
	const numChars int64 = 30
	switch {
	case c2 == ' ':
		hash = numChars * numChars * charValues[c1]
	case c3 == ' ':
		hash = numChars*numChars*charValues[c1] + numChars*charValues[c2]
	default:
		hash = numChars*numChars*charValues[c1] + numChars*charValues[c2] + charValues[c3]
	}
	return
}

// checkErr terminates program if any error has occurred.
func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}

// checkEOF returns true if err == io.EOF, false if err == nil, exits program if err is any other error.
func checkEOF(err error) bool {
	if err != nil {
		if err == io.EOF {
			return true
		} else {
			checkErr(err)
		}
	}
	return false
}
