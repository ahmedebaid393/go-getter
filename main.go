package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

type Download struct {
	Url           string
	TargetPath    string
	TotalSections int
}

func main() {
	startTime := time.Now()

	d := Download{
		Url:           "https://images.pexels.com/photos/145939/pexels-photo-145939.jpeg?cs=srgb&dl=pexels-flickr-145939.jpg&fm=jpg",
		TargetPath:    "image.jpg",
		TotalSections: 10,
	}

	err := d.Do()
	if err != nil {
		log.Fatalf("An error occurred. Error says: %s\n", err)
	}

	downloadTime := time.Now().Sub(startTime).Seconds()
	fmt.Printf("Download completed in %v seconds\n", downloadTime)

}

func (d Download) Do() error {
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
	fmt.Printf("Got %v\n", statusCode)

	if statusCode > 299 {
		return errors.New(fmt.Sprintf("Request status code is not valid, it is %v", statusCode))
	}

	size, err := strconv.Atoi(resp.Header.Get("Content-Length"))
	if err != nil {
		return err
	}
	fmt.Printf("Size is %v bytes\n", size)

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

	var wg sync.WaitGroup

	for i, s := range sections {
		wg.Add(1)
		// store variables as they change in looping
		i := i
		s := s

		// download sections concurrently
		go func() {
			defer wg.Done()
			err := d.downloadSection(i, s)
			if err != nil {
				panic(err)
			}
		}()
	}

	wg.Wait()

	err = d.mergeTmpFiles(sections)
	if err != nil {
		return err
	}

	return nil
}

func (d Download) getNewRequest(method string) (*http.Request, error) {
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

func (d Download) downloadSection(i int, s [2]int) error {
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

	err = ioutil.WriteFile(fmt.Sprintf("section-%v.tmp", i), b, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}

func (d Download) mergeTmpFiles(sections [][2]int) error {
	f, err := os.OpenFile(d.TargetPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModePerm)
	if err != nil {
		return err
	}

	defer f.Close()

	for i := range sections {
		b, err := ioutil.ReadFile(fmt.Sprintf("section-%v.tmp", i))
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

/*
* TODO - Improvements
* get file url user input
* validate file url
* get file name from url header
* get download folder path from user input or assume the current folder is it
 */
