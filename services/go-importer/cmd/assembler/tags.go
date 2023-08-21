package main

import (
	"go-importer/internal/pkg/db"
	"log"
	"regexp"
	"strings"
)

var flagRegex *regexp.Regexp

func EnsureRegex(reg *string) {
	if flagRegex == nil {
		reg, err := regexp.Compile(*reg)
		if err != nil {
			log.Fatal("Failed to compile flag regex: ", err)
		} else {
			flagRegex = reg
		}
	}
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// Apply flag in/flag out tags to the entire flow.
// This assumes the `Data` part of the flowItem is already pre-processed, s.t.
// we can run regex tags over the payload directly
// also add the matched flags to the FlowItem
func ApplyFlagTags(flow *db.FlowEntry, reg *string) {
	EnsureRegex(reg)

	// If the regex is not valid, bail here
	if flagRegex == nil {
		return
	}

	for idx := 0; idx < len(flow.Flow); idx++ {
		flowItem := &flow.Flow[idx]
		matches := flagRegex.FindAllStringSubmatch(flowItem.Data, -1)
		if len(matches) > 0 {
			var tag string
			if flowItem.From == "c" {
				tag = "flag-in"
			} else {
				tag = "flag-out"
			}

			// Add the flag if it doesn't already exist
			for _, match := range matches {
				var flag string
				flag = match[0]
				if !contains(flow.Flags, flag) {
					flow.Flags = append(flow.Flags, flag)
				}
			}

			// Add the tag if it doesn't already exist
			if !contains(flow.Tags, tag) {
				flow.Tags = append(flow.Tags, tag)
			}
		}
	}
}

// Apply flagids to the entire flow.
// This assumes the `Data` part of the flowItem is already pre-processed, s.t.
func ApplyFlagids(flow *db.FlowEntry, flagids []string) {

	for idx := 0; idx < len(flow.Flow); idx++ {
		flowItem := &flow.Flow[idx]
		data := flowItem.Data
		for _, flagid := range flagids {
			log.Print("DEBUG: " + flagid)
			if strings.Contains(data, flagid) {
				log.Print("DEBUG: Found Match")
				tag := "flagid"

				if !contains(flow.Flagids, flagid) {
					flow.Flagids = append(flow.Flagids, flagid)
				}

				// Add the tag if it doesn't already exist
				if !contains(flow.Tags, tag) {
					flow.Tags = append(flow.Tags, tag)
				}
			}
		}
	}
}
