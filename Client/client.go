package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"golang.org/x/net/html"
)

var wg sync.WaitGroup

// CheckedFile is defined to hold a file path and a file scanner.
type CheckedFile struct {
	filePath    string
	dataScanner *bufio.Scanner
}

// PerformGetRequest dispatches a GET request to the file server to get a list of files.
func PerformGetRequest() string {
	const url = "http://localhost:8080"

	response, err := http.Get(url)

	if err != nil {
		panic(err)
	}

	defer response.Body.Close()

	var responseString strings.Builder
	content, _ := ioutil.ReadAll(response.Body)
	byteCount, _ := responseString.Write(content)
	fmt.Println("ByteCount is: ", byteCount)
	return responseString.String()
}

// ParseHtml is used to pharse the HTML response we get from the servers.
// Assumes that the file name is stored within an <a></a> HTML tag.
func ParseHtml(text string) (data []string) {

	tkn := html.NewTokenizer(strings.NewReader(text))
	var vals []string
	var isLi bool

	for {
		tt := tkn.Next()
		switch {
		case tt == html.ErrorToken:
			return vals
		case tt == html.StartTagToken:
			t := tkn.Token()
			isLi = t.Data == "a"
		case tt == html.TextToken:
			t := tkn.Token()
			if isLi {
				vals = append(vals, t.Data)
			}
			isLi = false
		}
	}
}

// IsDirectory determines if a file represented
// by `path` is a directory or not
func IsDirectory(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		fmt.Println(err)
		return false
	}
	switch mode := fi.Mode(); {
	case mode.IsDir():
		return true
	}
	return false
}

func IsFileEmpty(path string) bool {
	fInfo, err := os.Stat(path)
	if err != nil {
		log.Fatal(err)
	}
	fsize := fInfo.Size()

	return fsize == 0
}

// FillCheckedFilesArray finds all the relevant files, builds a CheckedFile entity out of them,
// and groups them into an array
func FillCheckedFilesArray(filesList []string, allCheckedFiles *[]CheckedFile) {
	defer wg.Done()
	for _, item := range filesList {
		if IsDirectory("../"+item) || strings.Contains(item, ".go") {
			continue
		}
		filePath := "../" + item
		if IsFileEmpty(filePath) {
			continue
		}
		file, err := os.Open(filePath)
		if err != nil {
			log.Fatal(err)
		} else {
			fmt.Println(file)
		}
		data := bufio.NewScanner(file)
		data.Split(bufio.ScanRunes)
		*allCheckedFiles = append(*allCheckedFiles, CheckedFile{filePath: "../" + item, dataScanner: data})
	}
}

// GetAllPositions returns all the relevant indexes out of a word
// according to the specific letter we search
func GetAllPositions(word string, searchLetter string) []int {
	var allIndexes []int
	for i, x := range word {
		if string(x) == searchLetter {
			allIndexes = append(allIndexes, i)
		}
	}
	return allIndexes
}

// CalculateSearchWord builds an ouput word that is assembeled from all
// the input files (while reading them paralally).
// Empty char will be filled with a "-"
func CalculateSearchWord(allCheckedFiles []CheckedFile) string {
	output := ""
	for _, ent := range allCheckedFiles {
		success := ent.dataScanner.Scan()
		if success == false {
			err := ent.dataScanner.Err()
			if err == nil {
				continue
			} else {
				log.Fatal(err)
			}
		}
		letter := ent.dataScanner.Text()
		if letter == "" {
			letter = "-"
		}
		output += letter
	}
	return output
}

// CalculateOutputWord searches for the desired letter (in our case "A")
// and iterating all the input files
func CalculateOutputWord(allCheckedFiles []CheckedFile, output *string) {
	defer wg.Done()
	*output = ""
	for !strings.Contains(*output, "A") {
		*output = CalculateSearchWord(allCheckedFiles)
		if len(*output) == 0 {
			break
		}
	}
}

// FillFoundFilesList builds a list that being read of the server
func FillFoundFilesList(output string, allCheckedFiles []CheckedFile, foundFilesList *[]string) {
	if len(output) > 0 {
		positions := GetAllPositions(output, "A")
		for _, item := range positions {
			*foundFilesList = append(*foundFilesList, allCheckedFiles[item].filePath)
		}
	}
}

// DownloadFile downloads the file from the server to the current working folder
func DownloadFile(source string, destination string) {
	// Open original file
	original, err := os.Open(source)
	if err != nil {
		log.Fatal(err)
	}
	defer original.Close()

	// Create new file
	new, err := os.Create(destination + "\\" + strings.Split(original.Name(), "../")[1])
	if err != nil {
		log.Fatal(err)
	}
	defer new.Close()

	//This will copy
	bytesWritten, err := io.Copy(new, original)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Bytes Written: %d\n", bytesWritten)
}

// GetCurrentPath gives you the wordking folder
func GetCurrentPath() string {
	path, err := os.Getwd()
	if err != nil {
		log.Println(err)
	}
	return path
}

func GetFileListByRequest() []string {
	// Get all the files from the server
	htmlContent := PerformGetRequest()
	// Parse the response into a list
	filesList := ParseHtml(htmlContent)
	return filesList
}

// DownloadFiles downloads the relevant files only to the current WD
func DownloadFiles(foundFilesList []string) {
	for _, currFilePath := range foundFilesList {
		DownloadFile(currFilePath, GetCurrentPath())
	}
}

func main() {
	var allCheckedFiles []CheckedFile
	var foundFilesList []string
	var output string
	fmt.Println("Starting...")
	filesList := GetFileListByRequest()
	wg.Add(1)
	// Fetch the files from the server and store their pathes and their data scanners
	go FillCheckedFilesArray(filesList, &allCheckedFiles)
	wg.Wait()
	wg.Add(1)
	// Search for the first appearance "A" in one of the files / several files
	go CalculateOutputWord(allCheckedFiles, &output)
	wg.Wait()
	// Build a list of files, out of the files you found "A" in
	FillFoundFilesList(output, allCheckedFiles, &foundFilesList)
	if len(foundFilesList) > 0 {
		fmt.Println(foundFilesList)
	} else {
		fmt.Println("Could not find any A in any of the files")
	}
	DownloadFiles(foundFilesList)
	fmt.Println("Finished!")
}
