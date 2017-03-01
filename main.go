package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func getHashesFromFrontmatter(path string) {
	file, err := os.Open(path)
	check(err)
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Scan()
	delim := scanner.Text()

	var text string
	var tomlStr string
	for ; text != delim; text = scanner.Text() {
		scanner.Scan()
		if len(text) > 0 {
			tomlStr += text + "\n"
		}
	}
	//fmt.Print(tomlStr)

	type tomlStruct struct {
		Title      string
		Image      string
		ImageIpfs  string   `toml:"image_ipfs"`
		ImagesIpfs []string `toml:"images_ipfs"`
	}

	var tomlObj tomlStruct
	_, er := toml.Decode(tomlStr, &tomlObj)
	check(er)
	//fmt.Printf("%s (%s)\n", tomlObj.Title, tomlObj.Image)
	fmt.Println()
	fmt.Println(tomlObj.Title)
	if len(tomlObj.ImageIpfs) > 0 {
		fmt.Println(tomlObj.ImageIpfs)
		downloadFile("output/"+tomlObj.ImageIpfs+".jpg",
			"http://gateway.ipfs.io/ipfs/"+tomlObj.ImageIpfs)
	}
	for _, ipfs := range tomlObj.ImagesIpfs {
		fmt.Println(ipfs)
		downloadFile("output/"+ipfs+".jpg",
			"http://gateway.ipfs.io/ipfs/"+ipfs)
	}
}

func timeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Printf("%s took %s", name, elapsed)
}

// WriteCounter counts the number of bytes written to it.
type WriteCounter struct {
	Total int64 // Total # of bytes transferred
}

// Write implements the io.Writer interface.
//
// Always completes and never returns an error.
func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Total += int64(n)
	fmt.Printf("\rRead %d bytes for a total of %d\n", n, wc.Total)
	return n, nil
}

func downloadFile(filepath string, url string) (err error) {
	if _, er := os.Stat(filepath); !os.IsNotExist(er) {
		fmt.Println("File already exists. Continue...")
		return er
	}
	defer timeTrack(time.Now(), "downloading "+url)

	// Get the data
	timeout := time.Duration(35 * time.Second)
	client := http.Client{
		Timeout: timeout,
	}
	resp, err := client.Get(url)
	if err != nil {
		fmt.Print(err)
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		fmt.Print(err)
		return err
	}
	defer out.Close()
	// Writer the body to file
	src := io.TeeReader(resp.Body, &WriteCounter{})
	_, err = io.Copy(out, src)
	if err != nil {
		fmt.Print("Copy error:", err)
		return err
	}

	return nil
}

func main() {

	os.Mkdir("output", os.ModePerm)
	dirPath := os.Args[1]

	c := make(chan error)
	go func() {
		c <- filepath.Walk(dirPath,
			func(path string, _ os.FileInfo, _ error) error {
				if strings.HasSuffix(strings.ToLower(path), ".md") {

					getHashesFromFrontmatter(path)
				}
				return nil
			})
	}()
	<-c
}
