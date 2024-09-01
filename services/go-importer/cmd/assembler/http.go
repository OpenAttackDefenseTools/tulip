package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"go-importer/internal/pkg/db"
	"hash/crc32"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/andybalholm/brotli"
)

func AddFingerprints(cookies []*http.Cookie, fingerPrints map[uint32]bool) {
	for _, cookie := range cookies {

		// Prevent exploitation by encoding :pray:, who cares about collisions
		checksum := crc32.Checksum([]byte(url.QueryEscape(cookie.Name)), crc32.IEEETable)
		checksum = crc32.Update(checksum, crc32.IEEETable, []byte("="))
		checksum = crc32.Update(checksum, crc32.IEEETable, []byte(url.QueryEscape(cookie.Value)))
		fingerPrints[checksum] = true
	}
}

// Parse and simplify every item in the flow. Items that were not successfuly
// parsed are left as-is.
//
// If we manage to simplify a flow, the new data is placed in flowEntry.data
func ParseHttpFlow(g_db *db.Database, flow *db.FlowEntry) {
	// Use a set to get rid of duplicates
	fingerprintsSet := make(map[uint32]bool)

	for i := range flow.Flow {
		flowItem := &flow.Flow[i]
		// Run only on raw representation
		if flowItem.Kind != "raw" {
			continue
		}

		// TODO; rethink the flowItem format to make this less clunky
		reader := bufio.NewReader(bytes.NewReader(flowItem.Data))

		if flowItem.From == "c" {
			// HTTP Request
			req, err := http.ReadRequest(reader)
			if err != nil || req == nil {
				continue
			}

			if !contains(flow.Tags, "http") {
				flow.Tags = append(flow.Tags, "http")
			}

			if *http_session_tracking {
				// Parse cookie and grab fingerprints
				AddFingerprints(req.Cookies(), fingerprintsSet)
			}

			//TODO; replace the HTTP data.
			// Remember to use a `LimitReader` when implementing this to prevent
			// decompressions bombs / DOS!
		} else if flowItem.From == "s" {
			// Parse HTTP Response
			res, err := http.ReadResponse(reader, nil)
			if err != nil || res == nil {
				continue
			}

			if !contains(flow.Tags, "http") {
				flow.Tags = append(flow.Tags, "http")
			}

			if *http_session_tracking {
				// Parse cookie and grab fingerprints
				AddFingerprints(res.Cookies(), fingerprintsSet)
			}

			// Substitute body
			encoding := res.Header["Content-Encoding"]
			if encoding == nil || len(encoding) == 0 {
				// If we don't find an encoding header, it is either not valid,
				// or already in plain text. In any case, we don't have to edit anything.
				continue
			}

			var newReader io.Reader
			if err != nil {
				// Failed to fully read the body. Bail out here
				continue
			}

			switch encoding[0] {
			case "gzip":
				newReader, err = handleGzip(res.Body)
				break
			case "br":
				newReader, err = handleBrotili(res.Body)
				break
			case "deflate":
				//TODO; verify this is correct
				newReader, err = handleGzip(res.Body)
				break
			default:
				// Skipped, unknown or identity encoding
				continue
			}

			// Replace the reader to allow for in-place decompression
			if err == nil && newReader != nil {
				// Limit the reader to prevent potential decompression bombs
				res.Body = io.NopCloser(io.LimitReader(newReader, int64(*maxFlowItemSize * 1024 * 1024)))
				// Delete the content-encoding header as we've basically skipped its purpose (otherwise, pkappa converters will have issues as they think it's still encoded).
				// In case of multiple values there, this logic wouldn't be hit anyway
				res.Header.Del("Content-Encoding")
				// invalidate the content length, since decompressing the body will change its value.
				res.ContentLength = -1
				replacement, err := httputil.DumpResponse(res, true)
				if err != nil {
					// HTTPUtil failed us, continue without replacing anything.
					continue
				}
				// This can exceed the mongo document limit, so we need to make sure
				// the replacement will fit
				new_size := flow.Size + (len(replacement) - len(flowItem.Data))
				if new_size <= *maxFlowItemSize * 1024 * 1024 {
					flowItem.Data = replacement
					flow.Size = new_size
				}
			}
		}
	}

	if *http_session_tracking {
		// Use maps.Keys(fingerprintsSet) in the future
		flow.Fingerprints = make([]uint32, 0, len(fingerprintsSet))
		for k := range fingerprintsSet {
			flow.Fingerprints = append(flow.Fingerprints, k)
		}
	}
}

func handleGzip(r io.Reader) (io.Reader, error) {
	return gzip.NewReader(r)
}

func handleBrotili(r io.Reader) (io.Reader, error) {
	reader := brotli.NewReader(r)
	return reader, nil
}
