package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v3"
)

// Morse code map
var morseCode = map[rune]string{
	'a': ".-", 'b': "-...", 'c': "-.-.", 'd': "-..", 'e': ".",
	'f': "..-.", 'g': "--.", 'h': "....", 'i': "..", 'j': ".---",
	'k': "-.-", 'l': ".-..", 'm': "--", 'n': "-.", 'o': "---",
	'p': ".--.", 'q': "--.-", 'r': ".-.", 's': "...", 't': "-",
	'u': "..-", 'v': "...-", 'w': ".--", 'x': "-..-", 'y': "-.--",
	'z': "--..",
}

func textToMorse(text string) string {
	var result []string
	words := strings.Split(strings.ToLower(text), " ")
	for _, word := range words {
		var wordSymbols []string
		for _, char := range strings.ToLower(word) {
			code, ok := morseCode[char]
			if !ok { // if character is not in morseCode
				continue
			}
			wordSymbols = append(wordSymbols, code)
		}
		result = append(result, strings.Join(wordSymbols, " "))
	}
	return strings.Join(result, "/")
}

// Helper function to write a MIDI variable-length quantity
func writeVarLength(value int) []byte {
	buf := []byte{}
	buffer := value & 0x7F
	for value >>= 7; value > 0; value >>= 7 {
		buffer <<= 8
		buffer |= ((value & 0x7F) | 0x80)
	}
	for {
		buf = append(buf, byte(buffer))
		if buffer&0x80 != 0 {
			buffer >>= 8
		} else {
			break
		}
	}
	return buf
}

const microsecondsPerMinute = 60000000

func createMIDI(morse string, filename string, bpm int) error {
	var track []byte

	// Set tempo using BPM
	microsecondsPerQuarterNote := microsecondsPerMinute / bpm
	track = append(track, 0x00, 0xFF, 0x51, 0x03,
		byte((microsecondsPerQuarterNote>>16)&0xFF),
		byte((microsecondsPerQuarterNote>>8)&0xFF),
		byte(microsecondsPerQuarterNote&0xFF),
	)

	ticksPerBeat := 96
	dot := ticksPerBeat / 2
	dash := dot * 3
	pause := dot * 7

	note := 76 // E5
	velocity := 100
	time := 0

	addNote := func(duration int) {
		track = append(track, writeVarLength(time)...)
		track = append(track, 0x90, byte(note), byte(velocity))
		track = append(track, writeVarLength(duration)...)
		track = append(track, 0x80, byte(note), 0x00)
	}

	for _, symbol := range morse {
		switch symbol {
		case '.':
			addNote(dot)
			time = dot
		case '-':
			addNote(dash)
			time = dot
		case ' ':
			time = 4 * dot
		case '/':
			time = pause
		}
	}

	// End of track
	track = append(track, writeVarLength(time)...)
	track = append(track, 0xFF, 0x2F, 0x00)

	// Build header and track chunk
	header := []byte{
		'M', 'T', 'h', 'd',
		0x00, 0x00, 0x00, 0x06,
		0x00, 0x00, // format
		0x00, 0x01, // one track
		0x00, byte(ticksPerBeat),
	}
	trackHeader := []byte{
		'M', 'T', 'r', 'k',
	}
	trackLength := make([]byte, 4)
	binary.BigEndian.PutUint32(trackLength, uint32(len(track)))

	// Combine all chunks
	data := append(header, trackHeader...)
	data = append(data, trackLength...)
	data = append(data, track...)

	// Write to file
	return os.WriteFile(filename, data, 0644)
}

func main() {
	cmd := &cli.Command{
		Name:  "go-morse-midi",
		Usage: "Convert text to Morse code MIDI",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:  "bpm",
				Value: 120,
				Usage: "tempo in beats per minute",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			text := strings.Join(c.Args().Slice(), " ")
			fmt.Println("Input text:", text)
			if strings.TrimSpace(text) == "" {
				return fmt.Errorf("no text provided, please provide text to convert to Morse code")
			}
			morse := textToMorse(text)
			fmt.Println("Morse code:", morse)

			outputFile := strings.ReplaceAll(text, " ", "-") + ".mid"
			bpm := c.Int("bpm")
			err := createMIDI(morse, outputFile, bpm)
			if err != nil {
				fmt.Println("Error writing MIDI file:", err)
			} else {
				fmt.Printf("MIDI file saved as %s\n", outputFile)
			}
			return err
		},
	}

	err := cmd.Run(context.TODO(), os.Args)
	if err != nil {
		fmt.Println("Error:", err)
		fmt.Println("Use --help for more information.")
		os.Exit(1)
	}
}
