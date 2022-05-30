package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"go-importer/internal/pkg/db"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/andybalholm/brotli"
)

// Parse and simplify every item in the flow. Items that were not successfuly
// parsed are left as-is.
//
// If we manage to simplify a flow, the new data is placed in flowEntry.data
func ParseHttpFlow(flow *db.FlowEntry) {
	for idx := 0; idx < len(flow.Flow); idx++ {
		flowItem := &flow.Flow[idx]
		// TODO; rethink the flowItem format to make this less clunky
		reader := bufio.NewReader(strings.NewReader(flowItem.Data))
		if flowItem.From == "c" {
			// HTTP Request
			_, err := http.ReadRequest(reader)
			if err == nil {
				//TODO; replace the HTTP data.
			}

		} else if flowItem.From == "s" {
			// Parse HTTP Response
			res, err := http.ReadResponse(reader, nil)
			if err != nil {
				log.Println("Skipped, not valid http")
				continue
			}
			// Substitute body

			encoding := res.Header["Content-Encoding"]
			if encoding == nil || len(encoding) == 0 {
				log.Println("Skipped, no content header")
				continue
			}

			var newReader io.ReadCloser
			body, err := ioutil.ReadAll(res.Body)
			r := bytes.NewBuffer(body)
			res.Body.Close()
			if err != nil {
				log.Println("Skipped, failed to read res")
				continue
			}
			switch encoding[0] {
			case "gzip":
				newReader, err = handleGzip(r)
				break
			case "br":
				newReader, err = handleBrotili(r)
				break
			case "deflate":
				newReader, err = handleGzip(r)
				break
			default:
				log.Println("Skipped, unknown encoding")
				continue
			}

			res.Body = newReader
			defer res.Body.Close()
			// invalidate the content length, since decompressing the body will change its value.
			res.ContentLength = -1
			replacement, err := httputil.DumpResponse(res, true)
			if err != nil {
				log.Println("Skipped, dumping failed")
				continue
			}

			flowItem.Data = string(replacement)
		}
	}
}

func handleGzip(r io.Reader) (io.ReadCloser, error) {
	reader, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	return reader, nil
}

func handleBrotili(r io.Reader) (io.ReadCloser, error) {
	reader := brotli.NewReader(r)
	ret := ioutil.NopCloser(reader)
	return ret, nil
}

func handleDeflate(r io.Reader) (io.ReadCloser, error) {
	return nil, nil
}
