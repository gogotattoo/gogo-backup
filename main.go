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

	"github.com/gogotattoo/common/models"
	json "github.com/nwidger/jsoncolor"

	"github.com/BurntSushi/toml"
	"github.com/fatih/color"
)

const (
	ipfsHost      = "https://ipfs.io/ipfs/"
	outputDefault = "output/"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func extractTomlStr(r io.Reader) (string, error) {
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
	return tomlStr, nil
}

func timeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	color.Green("\n%s took %s\n\n", name, elapsed)
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

var existingFiles = 0
var downloadedFiles = 0
var proceedFiles = 0
var brokenFiles []string

func downloadFileFromIPFS(filepath, ipfsHash string) (err error) {
	if _, er := os.Stat(filepath); !os.IsNotExist(er) {
		existingFiles++
		color.Blue("File already exists. Continue...")
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
		color.New(color.FgRed).Add(color.Underline).Println(err)
		brokenFiles = append(brokenFiles, ipfsHash)
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		color.New(color.FgRed).Add(color.Underline).Println(err)
		brokenFiles = append(brokenFiles, ipfsHash)
		return err
	}
	defer out.Close()
	// Writer the body to file
	src := io.TeeReader(resp.Body, &WriteCounter{})
	_, err = io.Copy(out, src)
	if err != nil {
		color.New(color.FgRed).Add(color.Underline).Println(err)
		brokenFiles = append(brokenFiles, ipfsHash)
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

				tatTomlStr, fmEr := extractTomlStr(file)

				if fmEr != nil {
					log.Println()
					color.New(color.FgRed).Add(color.Underline).Println(fmEr.Error())
					return fmEr
				}
				//fmt.Print(tomlStr)
				var tat models.Artwork
				_, er := toml.Decode(tatTomlStr, &tat)
				if er != nil {
					log.Println()
					color.New(color.FgRed).Add(color.Underline).Println(er.Error())
					return er
				}
				//fmt.Printf("%s (%s)\n", tomlObj.Title, tomlObj.Image)
				fmt.Println() // Mix up foreground and background colors, create new mixes!
				color.New(color.FgRed).Add(color.BgWhite).Println(tat.Title)
				tatJSON, _ := json.MarshalIndent(tat, "", "")
				log.Println(string(tatJSON))
				color.New(color.Bold).Add(color.Italic).Println(tatTomlStr)
				filePrefix := strings.Replace(tat.MadeDate[:10], "-", ".", -1)
				filePrefix += " - " + tat.Title + " @" + tat.MadeAtShop
				os.Mkdir(outputDefault+filePrefix, os.ModePerm)
				filePrefix += "/" + filePrefix
				if len(tat.ImageIpfs) > 0 {
					fmt.Println(tat.ImageIpfs)
					if downloadFileFromIPFS(outputDefault+filePrefix+tat.ImageIpfs+".jpg", tat.ImageIpfs) == nil {
						downloadedFiles++
					}
				}
				for _, ipfsHash := range tat.ImagesIpfs {
					fmt.Println(ipfsHash)

					if nil == downloadFileFromIPFS(outputDefault+filePrefix+ipfsHash+".jpg", ipfsHash) {
						downloadedFiles++
					}
				}

				return nil
			})
	}()
	<-c

	color.Green("Downloaded files: ", downloadedFiles)
	color.Green("Existing files: ", existingFiles)
	color.Red("Failed files: ", len(brokenFiles))
}
