package main

import "math/rand"

func generateLogLine() []byte {
	lineLength := rand.Intn(990) + 10 // generate a line of log with a length between 10 and 1000
	line := make([]byte, lineLength)
	for i := 0; i < lineLength-1; i++ {
		c := byte(rand.Int31n(95) + 32) // generate a rune from the character with index 32 (space) to index 126 (~)
		line[i] = c
	}
	line[lineLength-1] = byte('\n') // add a carrage return at the end of the line
	return line
}
