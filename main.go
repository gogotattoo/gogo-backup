package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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
	fmt.Println(path)
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

func downloadFile(filepath string, url string) (err error) {

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Writer the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
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
