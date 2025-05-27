package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type LogBlock struct {
	Time time.Time
	Text string
}

type FilterConfig struct {
	Contains []string `yaml:"contains"`
}

var dateTemplate = "2006-01-02 15:04:05,000"

func main() {
	inputDir := flag.String("path", "./logs", "Path to directory with log files")
	outputFile := flag.String("out", "combined_sorted.log", "Output file name")
	filtersFilePath := flag.String("filters", UserHomeDir()+"/.config/logmixer/filters.yaml", "File with filtration rules")
	flag.Parse()

	var blocks []LogBlock
	filters := getFilters(*filtersFilePath)
	fmt.Println(filters)
	processFiles(inputDir, &blocks, filters)
	sortBlocksByTime(blocks)
	err := writeCombinedLogFile(outputFile, &blocks)
	if err == nil {
		fmt.Printf("Done. Written to %s\n", *outputFile)
	} else {
		fmt.Printf("Error when saving result file  %s\n. %v", *outputFile, err)
	}
}

// Walks recusively through the directory with log files
// Runs processing on each file
func processFiles(inputDir *string, blocks *[]LogBlock, filters FilterConfig) {
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

// Reads file and devides it into message blocks
// One block - one message
// The result of this function is a list of message blocks
func processFile(path string, blocks *[]LogBlock) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("cannot open file %s: %w", path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 16*1024*1024)
	var currentBlock strings.Builder
	var currentTime time.Time

	for scanner.Scan() {
		line := scanner.Text()
		if isLogStart(line) {
			line = addFilePathToLine(line, file.Name())

			if currentBlock.Len() > 0 {
				addBlockToCollection(blocks, currentTime, &currentBlock)
			}

			t, _ := time.Parse(dateTemplate, line[:len(dateTemplate)])
			currentTime = t
		}

		currentBlock.WriteString(line + "\n")

	}

	if currentBlock.Len() > 0 {
		addBlockToCollection(blocks, currentTime, &currentBlock)
	}

	return scanner.Err()
}

func getFilters(filterFilePath string) FilterConfig {
	var filters FilterConfig
	data := readFiltersConfigFile(filterFilePath)
	err := yaml.Unmarshal([]byte(data), &filters)
	if err != nil {
		log.Fatalf("cannot unmarshal data: %v", err)
	}
	return filters
}

func readFiltersConfigFile(filterFilePath string) string {

	yamlFile, err := os.ReadFile(filterFilePath)
	if err != nil {
		log.Printf("Error while filters config file reading   #%v ", err)
	}

	return string(yamlFile)
}

func UserHomeDir() string {
	if runtime.GOOS == "windows" {
		home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return home
	}
	return os.Getenv("HOME")
}

// Defines is line from a text file is
// a start of log message
// or it is a middle line of a message
func isLogStart(line string) bool {
	if len(line) < len(dateTemplate) {
		return false
	}
	_, err := time.Parse(dateTemplate, line[:len(dateTemplate)])
	return err == nil
}

// Adds file name after timestamp in log line
func addFilePathToLine(line string, filepath string) string {
	if len(line) > len(dateTemplate) {
		prefix := line[:len(dateTemplate)]

		pathChunks := strings.Split(filepath, "/")

		if len(pathChunks) > 1 {
			prefix += " [" + strings.Join(pathChunks[1:], "/") + "] "
		} else {
			prefix += " [" + filepath + "] "
		}
		return prefix + line[len(dateTemplate):]

	} else {
		return line
	}
}

// Creates a block with a timestamp and log message
// Adds block to the blocks collection
func addBlockToCollection(blocks *[]LogBlock, currentTime time.Time, currentBlock *strings.Builder) {
	*blocks = append(*blocks, LogBlock{
		Time: currentTime,
		Text: currentBlock.String(),
	})
	currentBlock.Reset()
}

// Sorts message block by time in ascending order
func sortBlocksByTime(blocks []LogBlock) {
	sort.Slice(blocks, func(i, j int) bool {
		return blocks[i].Time.Before(blocks[j].Time)
	})
}

// Write the message blocks collection to a file
func writeCombinedLogFile(outputFile *string, blocks *[]LogBlock) error {
	out, err := os.Create(*outputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot create output file: %v\n", err)
		return err
	}
	defer out.Close()

	writer := bufio.NewWriter(out)
	for _, b := range *blocks {
		writer.WriteString(b.Text)
	}
	writer.Flush()

	return nil
}
