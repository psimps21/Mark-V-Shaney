// Copyright 2011 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Generating random text: a Markov chain algorithm

Based on the program presented in the "Design and Implementation" chapter
of The Practice of Programming (Kernighan and Pike, Addison-Wesley 1999).
See also Computer Recreations, Scientific American 260, 122 - 125 (1989).

A Markov chain algorithm generates text by creating a statistical model of
potential textual suffixes for a given prefix. Consider this text:

	I am not a number! I am a free man!

Our Markov chain algorithm would arrange this text into this set of prefixes
and suffixes, or "chain": (This table assumes a prefix length of two words.)

	Prefix       Suffix

	"" ""        I
	"" I         am
	I am         a
	I am         not
	a free       man!
	am a         free
	am not       a
	a number!    I
	number! I    am
	not a        number!

To generate text using this table we select an initial prefix ("I am", for
example), choose one of the suffixes associated with that prefix at random
with probability determined by the input statistics ("a"),
and then create a new prefix by removing the first word from the prefix
and appending the suffix (making the new prefix is "am a"). Repeat this process
until we can't find any suffixes for the current prefix or we exceed the word
limit. (The word limit is necessary as the chain table may contain cycles.)

Our version of this program reads text from standard input, parsing it into a
Markov chain, and writes generated text to standard output.
The prefix and output lengths can be specified using the -prefix and -words
flags on the command-line.
*/
package main

import (
	"bufio"
	"fmt"
	"io"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
)

// Prefix is a Markov chain prefix of one or more words.
type Prefix []string

// String returns the Prefix as a string (for use as a map key).
func (p Prefix) String() string {
	return strings.Join(p, " ")
}

// Shift removes the first word from the Prefix and appends the given word.
func (p Prefix) Shift(word string) {
	copy(p, p[1:])
	p[len(p)-1] = word
}

// Chain contains a map ("chain") of prefixes to a list of suffixes.
// A prefix is a string of prefixLen words joined with spaces.
// A suffix is a single word. A prefix can have multiple suffixes.
type Chain struct {
	chain     map[string][]string
	prefixLen int
	freqTable map[string]map[string]int
}

// NewChain returns a new Chain with prefixes of prefixLen words.
func NewChain(prefixLen int) *Chain {
	return &Chain{make(map[string][]string), prefixLen, make(map[string]map[string]int)}
}

// Build reads text from the provided Reader and
// parses it into prefixes and suffixes that are stored in Chain.
func (c *Chain) Build(r io.Reader) {
	br := bufio.NewReader(r)
	startPrefix := make([]string, c.prefixLen)
	for i := range startPrefix {
		startPrefix[i] = "\"\""
	}
	var p Prefix = startPrefix

	// p := make(Prefix, c.prefixLen)
	for {
		var s string
		if _, err := fmt.Fscan(br, &s); err != nil {
			break
		}
		key := p.String()

		if val, ok := c.freqTable[key]; ok { // if prefix is in table
			val[s]++
		} else { // if prefix is not in table
			c.freqTable[key] = map[string]int{s: 1}
		}

		c.chain[key] = append(c.chain[key], s)
		p.Shift(s)
	}
}

// FileToFreqTable reads a file and adds content to a frequency table
func (c *Chain) FileToFreqTable(filename string) {
	openFile, err := os.Open(filename)
	if err != nil {
		panic("Could not open input file.")
	}
	scanner := bufio.NewScanner(openFile)
	scanner.Split(bufio.ScanWords)

	startPrefix := make([]string, c.prefixLen)
	for i := range startPrefix {
		startPrefix[i] = "\"\""
	}
	var p Prefix = startPrefix

	for scanner.Scan() {
		s := scanner.Text()
		key := p.String()

		if val, ok := c.freqTable[key]; ok { // if prefix is in table
			val[s]++
		} else { // if prefix is not in table
			c.freqTable[key] = map[string]int{s: 1}
		}

		c.chain[key] = append(c.chain[key], s)
		p.Shift(s)
	}
}

// PrintFreqTable prints a frequency table
func (c *Chain) PrintFreqTable() {
	for k, v := range c.freqTable {
		var sufCount []string
		for k2, v2 := range v {
			sufCount = append(sufCount, k2, strconv.Itoa(v2))
		}
		fmt.Println(k + " " + strings.Join(sufCount, " "))
	}
}

//PrintChain prints a chain
func (c *Chain) PrintChain() {
	for key := range c.chain {
		fmt.Println(key, strings.Join(c.chain[key], " "))
	}
}

//ChainToFile writes the contents of a chain to a file
func (c *Chain) FreqTableToFile(filepath string) {
	// Open file
	openFile, err := os.Create(filepath)
	if err != nil {
		panic("Could not create file from given file path")
	}
	defer openFile.Close()

	writer := bufio.NewWriter(openFile)
	fmt.Fprintln(writer, strconv.Itoa(c.prefixLen))

	// Sort frequence table
	var keys []string
	for k := range c.freqTable {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Write each line of the output file
	for _, key := range keys {
		var sufCount []string
		for k2, v2 := range c.freqTable[key] {
			sufCount = append(sufCount, k2, strconv.Itoa(v2))
		}
		fmt.Fprintln(writer, key+" "+strings.Join(sufCount, " "))
	}
	writer.Flush()
}

// FreqTableFromFile create a frequency table from an freqTable file
func FreqTableFromFreqFile(freqFile string) Chain {
	openFile, err := os.Open(freqFile)
	if err != nil {
		panic("Could not open frequency table file.")
	}
	scanner := bufio.NewScanner(openFile)

	c := NewChain(0)
	for scanner.Scan() {
		items := strings.Split(scanner.Text(), " ")
		if len(items) > 1 { // if not first line of file and line is not empty
			// set prefix for the line
			var p Prefix
			for i := 0; i < c.prefixLen; i++ {
				p = append(p, items[i])
			}
			key := p.String()
			c.freqTable[key] = make(map[string]int)

			// update prefix map with suffix frequencies
			for i := c.prefixLen; i < len(items); i = i + 2 {
				sfxFreq, err := strconv.Atoi(items[i+1])
				if err != nil {
					panic("Could not convert string to integer")
				}
				c.freqTable[key][items[i]] = sfxFreq
			}
		} else if len(items) == 1 { // if first line or empty line
			if len(items[0]) > 0 { // if ifrst line
				// Set the prefix length for the chain
				prefixLen, err := strconv.Atoi(items[0])
				if err != nil {
					panic("Could not convert string to integer")
				}
				c.prefixLen = prefixLen
			}
		}
	}
	return *c
}

// ChainFromFreqTable generates a chain from a frequency table
func (c *Chain) ChainFromFreqTable() {
	for k, v := range c.freqTable { // for prefix in frequency table
		var sfxs []string
		for k2, v2 := range v { // for suffix in suffix map
			// append the key to for chain based on frequency
			for i := 0; i < v2; i++ {
				sfxs = append(sfxs, k2)
			}
		}
		c.chain[k] = sfxs
	}
}

// Generate returns a string of at most n words generated from Chain.
func (c *Chain) Generate(n int) string {
	startPrefix := make([]string, c.prefixLen)
	for i := range startPrefix {
		startPrefix[i] = "\"\""
	}
	var p Prefix = startPrefix
	var words []string
	for i := 0; i < n; i++ {
		choices := c.chain[p.String()]
		if len(choices) == 0 {
			break
		}
		next := choices[rand.Intn(len(choices))]
		words = append(words, next)
		p.Shift(next)
	}
	return strings.Join(words, " ")
}

func main() {
	if len(os.Args) < 4 {
		panic("Command does not have enough arguments")
	}

	runType := os.Args[1]
	if runType == "generate" {
		freqFile := os.Args[2]
		n, err := strconv.Atoi(os.Args[3])
		if err != nil {
			panic("Could not convert integer to string")
		}
		if n < 0 {
			panic("Number of words must be positive")
		}

		c := FreqTableFromFreqFile(freqFile)
		c.ChainFromFreqTable()
		text := c.Generate(n) // Generate text.
		fmt.Println(text)     // Write text to standard output.

	} else if runType == "read" {
		prefixLen, err := strconv.Atoi(os.Args[2])
		if err != nil {
			panic("Could not convert integer to string")
		}
		if prefixLen < 1 {
			panic("Prefix length must be greater than 0")
		}
		outputFile := os.Args[3]

		c := NewChain(prefixLen)
		// read all input files
		for i := 4; i < len(os.Args); i++ {
			c.FileToFreqTable(os.Args[i])
		}
		// Save frequency table to output file
		c.FreqTableToFile(outputFile)

	} else {
		panic("Invalid word in program command")
	}
}
