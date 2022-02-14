package downloader

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"sync"
)

type Downloader struct {
	Url              string
	FileName         string
	TargetPath       string `default:"./downloads"`
	TotalSections    int    `default:"10"`
	TmpDirectoryPath string
}

func Do(url string, targetPath string, numberOfSections int) error {
	// validate data
	validatedData, err := validateProvidedData(url, targetPath, numberOfSections)
	if err != nil {
		return err
	}

	// get filename from URL
	r, _ := http.NewRequest("GET", url, nil)

	// convert number of sections to string based on type in it's struct
	s, _ := strconv.Atoi(validatedData["numberOfSections"])

	// assign data to Downloader struct
	d := Downloader{
		Url:           validatedData["url"],
		FileName:      path.Base(r.URL.Path),
		TargetPath:    validatedData["targetPath"],
		TotalSections: s,
	}

	// start downloading the resource
	err = d.startDownload()
	if err != nil {
		return err
	}

	// return final result
	return nil
}

func validateProvidedData(url string, targetPath string, numberOfSections int) (map[string]string, error) {
	if len(url) == 0 {
		return nil, errors.New("Invalid URL")
	}

	_, err := http.Get(url)
	if err != nil {
		return nil, errors.New("URL is not reachable")
	}

	if len(targetPath) == 0 {
		return nil, errors.New("Invalid target path")
	}

	stat, err := os.Stat(targetPath)
	if err != nil || !stat.IsDir() {
		return nil, errors.New("Target path is not a valid directory")
	}

	if numberOfSections < 1 {
		return nil, errors.New("Number of sections must be greater than zero")
	}

	validateddata := map[string]string{
		"url":              url,
		"targetPath":       targetPath,
		"numberOfSections": fmt.Sprintf("%v", numberOfSections),
	}

	return validateddata, nil
}

func (d Downloader) startDownload() error {
	fmt.Printf("Making connection\n")
	r, err := d.getNewRequest("HEAD")
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return err
	}

	statusCode := resp.StatusCode
	if statusCode > 299 {
		return errors.New(fmt.Sprintf("Request status code is not valid, it is %v", statusCode))
	}

	size, err := strconv.Atoi(resp.Header.Get("Content-Length"))
	if err != nil {
		return err
	}
	fmt.Printf("Size is %v bytes\n", size)

	// create temp directory to save dowloaded sections
	tmpDir, err := os.MkdirTemp("", "example")
	if err != nil {
		return err
	}
	d.TmpDirectoryPath = tmpDir

	// split file bytes sections into slices
	sections := d.splitFileSectionsToSlices(size)

	// download file bytes sections concurrently
	d.downloadSectionsConcurrently(sections)

	// merge temp files
	err = d.mergeTmpFiles(sections)
	if err != nil {
		return err
	}

	// clear temp directory
	os.RemoveAll(d.TmpDirectoryPath)

	// return final result
	return nil
}

func (d Downloader) getNewRequest(method string) (*http.Request, error) {
	r, err := http.NewRequest(
		method,
		d.Url,
		nil,
	)

	if err != nil {
		return nil, err
	}

	return r, err
}

func (d Downloader) splitFileSectionsToSlices(size int) [][2]int {
	var sections = make([][2]int, d.TotalSections)
	eachSize := size / d.TotalSections

	for i := range sections {
		if i == 0 {
			sections[i][0] = 0
		} else {
			sections[i][0] = sections[i-1][1] + 1
		}

		if i < d.TotalSections-1 {
			sections[i][1] = sections[i][0] + eachSize
		} else {
			sections[i][1] = size - 1
		}
	}

	return sections
}

func (d Downloader) downloadSectionsConcurrently(sections [][2]int) {
	var wg sync.WaitGroup

	for i, s := range sections {
		wg.Add(1)

		go func(sectionStart int, sectionEnd [2]int) {
			defer wg.Done()

			err := d.downloadSection(sectionStart, sectionEnd)
			if err != nil {
				panic(err)
			}
		}(i, s)
	}

	wg.Wait()
}

func (d Downloader) downloadSection(i int, s [2]int) error {
	r, err := d.getNewRequest("GET")
	if err != nil {
		return err
	}

	r.Header.Set("Range", fmt.Sprintf("bytes=%v-%v", s[0], s[1]))
	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return err
	}
	sectionLength := resp.Header.Get("Content-Length")
	fmt.Printf("Downloaded %v bytes from section %v: %v \n", sectionLength, i, s)

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	tmpFile := filepath.Join(d.TmpDirectoryPath, fmt.Sprintf("section-%v.tmp", i))
	err = ioutil.WriteFile(tmpFile, b, 0666)
	if err != nil {
		return err
	}

	return nil
}

func (d Downloader) mergeTmpFiles(sections [][2]int) error {
	path := d.TargetPath + "/" + d.FileName
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}

	defer f.Close()

	for i := range sections {
		tmpFile := filepath.Join(d.TmpDirectoryPath, fmt.Sprintf("section-%v.tmp", i))
		b, err := ioutil.ReadFile(tmpFile)
		if err != nil {
			return err
		}

		n, err := f.Write(b)
		if err != nil {
			return err
		}

		fmt.Printf("%v bytes merged\n", n)
	}

	return nil
}
