package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"go-importer/internal/pkg/db"
	"io"
	"io/ioutil"
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
				continue
			}
			// Substitute body

			encoding := res.Header["Content-Encoding"]
			if encoding == nil || len(encoding) == 0 {
				// If we don't find an encoding header, it is either not valid,
				// or already in plain text. In any case, we don't have to edit anything.
				continue
			}

			var newReader io.ReadCloser
			body, err := ioutil.ReadAll(res.Body)
			r := bytes.NewBuffer(body)
			res.Body.Close()
			if err != nil {
				// Failed to fully read the body. Bail out here
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
				//TODO; verify this is correct
				newReader, err = handleGzip(r)
				break
			default:
				// Skipped, unknown or identity encoding
				continue
			}

			res.Body = newReader
			defer res.Body.Close()
			// invalidate the content length, since decompressing the body will change its value.
			res.ContentLength = -1
			replacement, err := httputil.DumpResponse(res, true)
			if err != nil {
				// HTTPUtil failed us, continue without replacing anything.
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
