package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"time"
)

func FileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func CreateFile(name string) error {
	file, err := os.Create(name)
	if err != nil {
		return err
	}
	defer func() {
		file.Close()
	}()
	return nil
}

func uploadFromFile() {

	for {
		f, err := os.Open("logfile")
		if err != nil {
			continue
		}
		defer f.Close()

		// Start reading from the file with a reader.
		reader := bufio.NewReader(f)
		var line string
		for {
			line, err = reader.ReadString('\n')
			if err != nil && err != io.EOF {
				break
			}

			// Process the line here.
			fmt.Printf(" > Read %d characters\n", len(line))
			fmt.Printf(" > > %s\n", limitLength(line, 50))

			if err != nil {
				break
			}
		}
		if err != io.EOF {
			fmt.Printf(" > Failed with error: %v\n", err)
		}

		time.Sleep(60 * time.Second)
	}
}

func limitLength(s string, length int) string {
	if len(s) < length {
		return s
	}
	return s[:length]
}
