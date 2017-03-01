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

const (
	ipfsHost      = "https://ipfs.io/ipfs/"
	outputDefault = "output/"
)

type tattoo struct {
	ID              string   `json:"id"`
	Link            string   `json:"link,omitempty"`
	Title           string   `json:"title,omitempty"`
	MadeDate        string   `json:"tattoodate,omitempty" toml:"tattoodate"`
	PublishDate     string   `json:"date,omitempty"`
	Tags            []string `json:"tags,omitempty"`
	BodyParts       []string `json:"bodypart,omitempty"`
	ImageIpfs       string   `json:"image_ipfs" toml:"image_ipfs"`
	ImagesIpfs      []string `json:"images_ipfs,omitempty" toml:"images_ipfs"`
	LocationCity    string   `json:"made_at_city" toml:"location_city"`
	LocationCountry string   `json:"made_at_country" toml:"location_country"`
	MadeAtShop      string   `json:"made_at_shop,omitempty" toml:"made_at_shop"`
	DurationMin     int      `json:"duration_min"`
	Gender          string   `json:"gender"`
	Extra           string   `json:"extra"`
	Article         string   `json:"article"`
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func findTattosFromFrontmatter(r io.Reader) (tattoo, error) {
	scanner := bufio.NewScanner(r)
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

	var tat tattoo
	_, er := toml.Decode(tomlStr, &tat)
	if er != nil {
		return tat, er
	}
	//fmt.Printf("%s (%s)\n", tomlObj.Title, tomlObj.Image)
	fmt.Println()
	fmt.Println(tat.Title)
	return tat, nil
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
	//fmt.Printf("\rRead %d bytes for a total of %d\n", n, wc.Total)
	return n, nil
}

func downloadFileFromIPFS(filepath, ipfsHash string) (err error) {
	if _, er := os.Stat(filepath); !os.IsNotExist(er) {
		fmt.Println("File already exists. Continue...")
		return er
	}
	defer timeTrack(time.Now(), "downloading "+ipfsHash)

	// Get the data
	timeout := time.Duration(35 * time.Second)
	client := http.Client{
		Timeout: timeout,
	}
	resp, err := client.Get(ipfsHost + ipfsHash)
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
				if !strings.HasSuffix(strings.ToLower(path), ".md") {
					return nil
				}
				file, err := os.Open(path)
				check(err)
				defer file.Close()

				tat, _ := findTattosFromFrontmatter(file)

				filePrefix := strings.Replace(tat.MadeDate[:10], "-", ".", -1)
				filePrefix += " - " + tat.Title + " @" + tat.MadeAtShop
				os.Mkdir(outputDefault+filePrefix, os.ModePerm)
				filePrefix += "/" + filePrefix
				if len(tat.ImageIpfs) > 0 {
					fmt.Println(tat.ImageIpfs)
					downloadFileFromIPFS(outputDefault+filePrefix+tat.ImageIpfs+".jpg", tat.ImageIpfs)
				}
				for _, ipfsHash := range tat.ImagesIpfs {
					fmt.Println(ipfsHash)
					downloadFileFromIPFS(outputDefault+filePrefix+ipfsHash+".jpg", ipfsHash)
				}

				return nil
			})
	}()
	<-c
}
