package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type LogBlock struct {
	Time time.Time
	Text string
}

var dateLayout = "2006-01-02 15:04:05,000"

func main() {
	inputDir := flag.String("path", "./logs", "Path to directory with log files")
	outputFile := flag.String("out", "combined_sorted.log", "Output file name")
	flag.Parse()

	var blocks []LogBlock

	processFiles(inputDir, &blocks)
	sortBlocksByTime(blocks)
	writeCombinedLogFile(outputFile, &blocks)

	fmt.Printf("Done. Written to %s\n", *outputFile)
}

func processFiles(inputDir *string, blocks *[]LogBlock) {
	err := filepath.WalkDir(*inputDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if strings.HasSuffix(d.Name(), ".log") || strings.HasSuffix(d.Name(), ".txt") {
			fmt.Printf("Processing %s\n", path)
			return processFile(path, blocks)
		}
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Directory walk error: %v\n", err)
		os.Exit(1)
	}
}

func processFile(path string, blocks *[]LogBlock) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("cannot open file %s: %w", path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var currentBlock strings.Builder
	var currentTime time.Time

	for scanner.Scan() {
		line := scanner.Text()

		if isLogStart(line) {
			line = addFilePathToLine(line, file.Name())

			if currentBlock.Len() > 0 {
				addBlockToCollection(blocks, currentTime, &currentBlock)

			}

			t, _ := time.Parse(dateLayout, line[:len(dateLayout)])
			currentTime = t
		}

		currentBlock.WriteString(line + "\n")

	}

	if currentBlock.Len() > 0 {
		addBlockToCollection(blocks, currentTime, &currentBlock)
	}

	return scanner.Err()
}

// Defines is line from a text file is
// a start of log message
// or it is a middle line of a message
func isLogStart(line string) bool {
	if len(line) < len(dateLayout) {
		return false
	}
	_, err := time.Parse(dateLayout, line[:len(dateLayout)])
	return err == nil
}

func addFilePathToLine(line string, filepath string) string {
	if len(line) > len(dateLayout) {
		prefix := line[:len(dateLayout)]

		pathChunks := strings.Split(filepath, "/")

		if len(pathChunks) > 1 {
			prefix += " [" + strings.Join(pathChunks[1:], "/") + "] "
		} else {
			prefix += " [" + filepath + "] "
		}
		return prefix + line[len(dateLayout):]

	} else {
		return line
	}
}

func addBlockToCollection(blocks *[]LogBlock, currentTime time.Time, currentBlock *strings.Builder) {
	*blocks = append(*blocks, LogBlock{
		Time: currentTime,
		Text: currentBlock.String(),
	})
	currentBlock.Reset()
}

func sortBlocksByTime(blocks []LogBlock) {
	sort.Slice(blocks, func(i, j int) bool {
		return blocks[i].Time.Before(blocks[j].Time)
	})
}

func writeCombinedLogFile(outputFile *string, blocks *[]LogBlock) {
	out, err := os.Create(*outputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot create output file: %v\n", err)
		os.Exit(1)
	}
	defer out.Close()

	writer := bufio.NewWriter(out)
	for _, b := range *blocks {
		writer.WriteString(b.Text)
	}
	writer.Flush()
}
