/*
Author: Edgar
Description: download files from github with specified url, no need to download while repository
*/
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

const GITHUB = "https://github.com"
const CONTENT = "https://raw.githubusercontent.com"

var urlPattern = regexp.MustCompile(`<a class="js-navigation-open.*?".*?title="(.*?)".*?href="(.*?)".*?>`)
var repositoryPattern = regexp.MustCompile(`(/.*?/.*?/)blob/(.*$)`)

// command line args
var respositoryURL string
var path string

func init() {
	flag.StringVar(&respositoryURL, "url", "", "the url you want to grab")
	flag.StringVar(&path, "dl", "", "the directory you want to save files")
}

func main() {
	flag.Parse()
	if respositoryURL == "" {
		fmt.Println("please specify the github url!")
		return
	}
	if path == "" {
		path = getPath(respositoryURL)
	}
	var client http.Client
	var wg sync.WaitGroup
	start := time.Now()
	dl(client, respositoryURL, path, &wg)
	wg.Wait()
	fmt.Printf("total time: %.2f s\n", float64(time.Since(start))/float64(time.Second))
}

// get all file link and download it
func dl(client http.Client, url, path string, wg *sync.WaitGroup) {
	// if the path is not existed, then create it
	if !isExist(path) {
		os.MkdirAll(path, 0775)
	}
	// get html source
	html, err := getHtml(client, url)
	if err != nil {
		fmt.Printf("get html error: %s", err.Error())
		return
	}
	// find all file and directory link
	links := urlPattern.FindAllSubmatch(html, -1)
	for _, link := range links {
		// if is directory, we can do it recursively
		if isDir(link[2]) {
			dl(client, GITHUB+string(link[2]), filepath.Join(path, getPath(string(link[2]))), wg)
		} else {
			// download it if it is file
			rep := repositoryPattern.FindSubmatch(link[2])
			// rep[1] is the repositoryPattern path
			// rep[2] is the file path in the repositoryPattern
			wg.Add(1)
			go downloadFile(client, CONTENT+string(rep[1])+string(rep[2]), path, string(link[1]), wg)
		}
	}
}

// download file
func downloadFile(client http.Client, fileURL, path, filename string, wg *sync.WaitGroup) {
	defer wg.Done()
	fmt.Println("start to download: ", filename)

	resp, err := client.Get(fileURL)
	if err != nil {
		fmt.Printf("download file %s failed due to: %s\n", filename, err.Error())
		return
	}
	defer resp.Body.Close()
	var buff [1024]byte
	// ????????????
	file, err := os.Create(filepath.Join(path, filename))
	if err != nil {
		fmt.Printf("create file: %s error\n", filename)
		return
	}
	defer file.Close()
	// ????????????
	for {
		n, err := resp.Body.Read(buff[:])
		if err != nil {
			if err == io.EOF {
				file.Write(buff[:n])
				fmt.Println("Read finished")
				break
			}
			fmt.Println("error: ", err)
			// if failed delete this file
			os.Remove(filepath.Join(path, filename))
			return
		}
		file.Write(buff[:n])
	}
	fmt.Println("finish download:", filename)
}

// get html source
func getHtml(client http.Client, url string) ([]byte, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// if is a directory
func isDir(link []byte) bool {
	return bytes.Contains(link, []byte("tree"))
}

// if file or directory exits
func isExist(path string) bool {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return true
}

func getPath(responsitoryUrl string) string {
	tmp := strings.TrimRight(responsitoryUrl, "/")
	i := strings.LastIndex(tmp, "/")
	return tmp[i+1:]
}
