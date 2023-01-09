/*
 * Authors: Leo Vainio, lvainio@kth.se &
 */

// testord: 	i (vanligaste ordet i svenskan),
//			 	öööö (sista ordet i rawindex.txt),
//				a (första i rawindex.txt),
//				amager (första ordet i korpus),
//				mult (sista ordet i korpus),
// 				the (vanligaste engelska ordet)
//				öööööööööööööööööööööö
//              en (tvåbokstavsord och väldigt vanligt ord)

package main

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bjarneh/latinx"
)

var (
	korpusPath = "/afs/kth.se/misc/info/kurser/DD2350/adk22/labb1/korpus"
	korpusSize int64
	korpus     *os.File
)

var (
	rawIndexPath = "/afs/kth.se/misc/info/kurser/DD2350/adk22/labb1/rawindex.txt"
	rawIndexSize int64
	rawIndex     *os.File
)

var (
	converter  = latinx.Get(latinx.Windows1252) // extended latin-1
	charValues map[byte]int64
)

var start time.Time

// init Creates a map with each letters value.
func init() {
	alphabet, _, err := converter.Encode([]byte(" abcdefghijklmnopqrstuvwxyzäåö"))
	checkErr(err)

	// Give characters a, b, ..., ö values 1, 2, ..., 28
	charValues = make(map[byte]int64)
	for k, c := range alphabet {
		charValues[c] = int64(k)
	}

	// Open korpus
	korpus, err = os.Open(korpusPath)
	checkErr(err)
	s, err := korpus.Stat()
	checkErr(err)
	korpusSize = s.Size()

	// Open rawIndex.txt
	rawIndex, err = os.Open(rawIndexPath)
	checkErr(err)
	s, err = rawIndex.Stat()
	checkErr(err)
	rawIndexSize = s.Size()
}

// main
func main() {
	word := strings.ToLower(getInput())
	start = time.Now()
	ptr := binarySearch(word)
	printResult(ptr, word)

	korpus.Close()
	rawIndex.Close()
}

// getSearchWord returns word as the command line argument or asks the user to enter a search word and return that.
func getInput() string {
	args := os.Args[1:]
	if len(args) > 1 {
		fmt.Println("Usage: korpus <word>")
		os.Exit(1)
	}
	word := ""
	if len(args) == 1 {
		word = args[0]
	} else {
		fmt.Print("Enter a search word: ")
		fmt.Scanln(&word)
	}
	return word
}

// binarySearch returns a pointer to the first occurrence of word in rawindex.txt.
func binarySearch(word string) int64 {
	br := bufio.NewReader(rawIndex)

	// ----- BINARY SEARCH ----- //
	word = strings.TrimSpace(word)
	hash := hash(word)
	left, right := getBSPointers(hash)
	for right-left > 1000 {
		// Go to middle.
		mid := (left + right) / 2
		_, err := rawIndex.Seek(mid, io.SeekStart)
		checkErr(err)

		// Go to new line.
		b := make([]byte, 1)
		for b[0] != '\n' {
			_, err := rawIndex.Read(b)
			checkErr(err)
		}

		// Read in word.
		br.Reset(rawIndex)
		w, err := br.ReadBytes(' ')
		checkErr(err)

		// Compare the words
		utf8, err := converter.Decode(w[:len(w)-1])
		checkErr(err)
		if string(utf8) < word {
			left = mid
		} else {
			right = mid
		}
	}

	// Make sure we are at the beginning of a new line before starting linear search.
	_, err := rawIndex.Seek(left, io.SeekStart)
	checkErr(err)
	if left > 0 {
		p := left - 1
		_, err = rawIndex.Seek(p, io.SeekStart)
		checkErr(err)
		b := make([]byte, 1)
		for b[0] != '\n' {
			_, err = rawIndex.Read(b)
			checkErr(err)
		}
	}

	// ----- LINEAR SEARCH ----- //
	br.Reset(rawIndex)
	ptr, err := rawIndex.Seek(0, io.SeekCurrent)
	checkErr(err)
	for {
		// Read in word.
		s, err := br.ReadBytes(' ')
		if checkEOF(err) {
			fmt.Println("Det finns 0 förekomster av ordet!")
			os.Exit(0)
		}

		sutf8, err := converter.Decode(s)
		checkErr(err)

		// If we have found our word return the pointer.
		if string(sutf8[:len(sutf8)-1]) == word {
			return ptr
		} else if string(sutf8) > word {
			fmt.Println("Det finns 0 förekomster av ordet!")
			os.Exit(0)
		}

		// Go to new line.
		b, err := br.ReadBytes('\n')
		checkErr(err)

		ptr += int64(len(s))
		ptr += int64(len(b))
	}
}

// hash hashes the first three letters of a word, each possible three letter combination returns a unique hash
// from 0 (aaa, aa, a) to 29^3-1 (ööö)
func hash(word string) (hash int64) {
	word = strings.TrimSpace(word)
	l1, _, err := converter.Encode([]byte(word))
	checkErr(err)
	const numChars int64 = 30
	wordLen := len(l1)
	switch {
	case wordLen >= 3:
		hash = numChars*numChars*charValues[l1[0]] + numChars*charValues[l1[1]] + charValues[l1[2]]
	case wordLen == 2:
		hash = numChars*numChars*charValues[l1[0]] + numChars*charValues[l1[1]]
	case wordLen == 1:
		hash = numChars * numChars * charValues[l1[0]]
	default:
		fmt.Println("Can't hash this!")
		os.Exit(1)
	}
	return
}

// getBSPointers returns hash corresponding pointer and the pointer to the next hash in rawIndex.txt that will
// be used for binary search.
func getBSPointers(hash int64) (left, right int64) {
	index, err := os.Open("../index/index")
	checkErr(err)
	_, err = index.Seek(hash*8, io.SeekStart)
	checkErr(err)
	err = binary.Read(index, binary.LittleEndian, &left)
	checkErr(err)

	const maxHash = 30 * 30 * 30
	var nextHash int64 = hash + 1
	if nextHash < maxHash {
		_, err = index.Seek(nextHash*8, io.SeekStart)
		checkErr(err)
		err = binary.Read(index, binary.LittleEndian, &right)
		checkErr(err)
	} else {
		right = rawIndexSize
	}

	// Can't do binary search on 0 bytes.
	if left == right {
		fmt.Println("Det finns 0 förekomster av ditt ordet")
		os.Exit(0)
	}

	index.Close()
	return
}

// printResult prints all occurrences from korpus of the input word.
func printResult(ptr int64, word string) {
	word = strings.TrimSpace(word)
	first := ptr
	rawIndex.Seek(ptr, 0)
	br := bufio.NewReader(rawIndex)

	count := 0
	for {
		// Read in a word.
		w, err := br.ReadBytes(' ')
		if checkEOF(err) {
			break
		}
		utf8w, _ := converter.Decode(w)

		// Break if we have reached a word we are not looking for.
		if string(utf8w[:len(utf8w)-1]) != word {
			break
		}

		// Go to next line
		br.ReadBytes('\n')
		count++
	}

	duration := time.Since(start)
	fmt.Println("Det finns", count, "förekomster av ordet.", "Search took:", duration)

	if count > 25 {
		fmt.Println("Tryck på någon tangent för att visa förekomsterna")
		fmt.Scanln()
	}

	rawIndex.Seek(first, io.SeekStart)
	br.Reset(rawIndex)
	for i := 0; i < count; i++ {
		// Read in a word.
		_, err := br.ReadBytes(' ')
		if checkEOF(err) {
			break
		}

		// Go to new line
		p, _ := br.ReadBytes('\n')
		ptr, _ := strconv.ParseInt(string(p[:len(p)-1]), 10, 64)
		printResLn(ptr, int64(len(word)))
	}
}

// printResult prints the word and adjacent text (before and after the word) in korpus at given position.
// Returns error if file pointer is out of bounds (fptr < 0 or fptr > korpusSize).
func printResLn(fptr int64, wordLen int64) error {
	var offset int64 = 30
	var numBytes int64
	if fptr < 0 || fptr > korpusSize-wordLen {
		return errors.New("index out of bounds (korpus)")
	} else if fptr < offset {
		numBytes = fptr + wordLen + offset
		fptr = 0
	} else {
		numBytes = offset*2 + wordLen
		fptr = fptr - offset
	}

	_, err := korpus.Seek(fptr, 0)
	checkErr(err)

	l1bytes := make([]byte, numBytes)
	_, err = korpus.Read(l1bytes)
	checkErr(err)

	utf8bytes, _ := converter.Decode(l1bytes)
	res := strings.Replace(string(utf8bytes), "\n", " ", -1)
	fmt.Println(string(res))
	return nil
}

// checkErr checks whether an error has occurred. Exit program if it has, does nothing if no error has occurred.
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
