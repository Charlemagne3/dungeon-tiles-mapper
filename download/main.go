package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
)

func main() {

	f, err := os.Open("./urls.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	var urls []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		urls = append(urls, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	re := regexp.MustCompile(`(DT[0-9])/(.+\.(jpg|gif)$)`)

	config := &tls.Config{
		InsecureSkipVerify: true,
	}
	tr := &http.Transport{TLSClientConfig: config}
	client := &http.Client{Transport: tr}

	for _, v := range urls {
		matches := re.FindStringSubmatch(v)
		if len(matches) > 2 {
			res, err := client.Get(v)
			if err != nil {
				log.Fatal(err)
			}
			if res.StatusCode != http.StatusOK {
				log.Fatal(res.StatusCode)
			}
			defer res.Body.Close()

			f, err := os.Create(fmt.Sprintf("%s_%s", matches[1], matches[2]))
			if err != nil {
				log.Fatal(err)
			}
			defer f.Close()

			_, err = io.Copy(f, res.Body)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}
