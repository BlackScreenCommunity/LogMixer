package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

type LogBlock struct {
	Time time.Time
	Text string
}

type FilterConfig struct {
	Exclude []string `yaml:"exclude"`
	Include []string `yaml:"include"`
}

var dateTemplate = "2006-01-02 15:04:05,000"
var count = 0

func main() {
	start := time.Now()
	inputDir, outputFile, filtersFilePath := prepareCommandLineArguments()

	err := ProcessLogFiles(filtersFilePath, inputDir, outputFile)

	if err != nil {
		fmt.Printf("Error when saving result file  %s\n. %v", *outputFile, err)
	}

	elapsed := time.Since(start)
	fmt.Printf("Time took %s ", elapsed)
	fmt.Printf("Done. Written to %s\n", *outputFile)
}

func prepareCommandLineArguments() (*string, *string, *string) {
	inputDir := flag.String("path", "./logs", "Path to directory with log files")
	outputFile := flag.String("out", "combined_sorted.log", "Output file name")

	currentDirectory, _ := os.Getwd()
	filtersFilePath := flag.String("filters", currentDirectory+"/filters.yaml", "File with filtration rules")

	flag.Parse()
	return inputDir, outputFile, filtersFilePath
}

func ProcessLogFiles(filtersFilePath *string, inputDir *string, outputFile *string) error {
	var blocks []LogBlock
	filters := getFilters(*filtersFilePath)
	processFiles(inputDir, &blocks, filters)
	sortBlocksByTime(blocks)
	err := writeCombinedLogFile(outputFile, &blocks)
	return err
}

// Walks recursively through the directory with log files
// Runs processing on each file
func processFiles(inputDir *string, blocks *[]LogBlock, filters FilterConfig) {
	var wg sync.WaitGroup

	err := filepath.WalkDir(*inputDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if strings.HasSuffix(d.Name(), ".log") || strings.HasSuffix(d.Name(), ".txt") {
			wg.Add(1)

			fmt.Printf("Processing %s\n", path)
			processFile(path, blocks, filters, &wg)
		}

		wg.Wait()
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Directory walk error: %v\n", err)
		os.Exit(1)
	}
}

// Reads file and divides it into message blocks
// One block - one message
// The result of this function is a list of message blocks
func processFile(path string, blocks *[]LogBlock, filters FilterConfig, wg *sync.WaitGroup) error {

	defer wg.Done()

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
				if !getIsBlockNeedsToFilter(currentBlock, filters) {
					addBlockToCollection(blocks, currentTime, &currentBlock)
				} else {
					currentBlock.Reset()
				}
			}

			t, _ := time.Parse(dateTemplate, line[:len(dateTemplate)])
			currentTime = t
		}

		currentBlock.WriteString(line + "\n")

	}

	if currentBlock.Len() > 0 {
		if !getIsBlockNeedsToFilter(currentBlock, filters) {
			addBlockToCollection(blocks, currentTime, &currentBlock)
		}
	}

	return scanner.Err()
}

// Checks if the current log block
// contains any of the filter phrases
func getIsBlockNeedsToFilter(currentBlock strings.Builder, filters FilterConfig) bool {
	if len(filters.Exclude) == 0 && len(filters.Include) == 0 {
		return false
	}

	blockString := currentBlock.String()

	if len(filters.Exclude) > 0 && len(filters.Include) == 0 {
		return getIsCollectionContainsString(filters.Exclude, blockString)
	}

	if len(filters.Exclude) > 0 && len(filters.Include) > 0 {
		if !getIsCollectionContainsString(filters.Include, blockString) {
			return true
		} else {
			return getIsCollectionContainsString(filters.Exclude, blockString)
		}
	}

	if len(filters.Exclude) == 0 && len(filters.Include) > 0 {
		result := !getIsCollectionContainsString(filters.Include, blockString)
		if result {
			return result
		}
	}

	return false
}

func getIsCollectionContainsString(stringCollection []string, substring string) bool {
	for _, filterString := range stringCollection {

		if strings.Contains(substring, filterString) {
			return true
		}
	}
	return false
}

// Reads filters data from yaml file
// And deserializes to a filters collection
func getFilters(filterFilePath string) FilterConfig {
	var filters FilterConfig
	data := readFile(filterFilePath)
	err := yaml.Unmarshal([]byte(data), &filters)
	if err != nil {
		log.Fatalf("cannot unmarshal data: %v", err)
	}
	return filters
}

// Reads file from file system
func readFile(filterFilePath string) string {
	log.Printf("Trying to find and open filters file in directory: %s", filterFilePath)

	fileContent, err := os.ReadFile(filterFilePath)
	if err != nil {
		log.Printf("Error while filters config file reading   #%v ", err)
	}

	return string(fileContent)
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
