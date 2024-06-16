package main

import (
	"go-importer/internal/pkg/db"

	"log"
	"regexp"

	"github.com/cloudflare/ahocorasick"
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
func ApplyFlagTags(flow *db.FlowEntry, reg *string, flagValidator FlagValidator) {
	EnsureRegex(reg)

	// If the regex is not valid, bail here
	if flagRegex == nil {
		return
	}

	flagsIn := 0
	flagsOut := 0
	for idx := 0; idx < len(flow.Flow); idx++ {
		flowItem := &flow.Flow[idx]
		matches := flagRegex.FindAll(flowItem.Data, -1)

		if len(matches) > 0 {
			var tags []string
			if flowItem.From == "c" {
				tags = append(tags, "flag-in")
				if len(matches) > flagsIn {
					flagsIn = len(matches)
				}
			} else {
				tags = append(tags, "flag-out")
				if len(matches) > flagsOut {
					flagsOut = len(matches)
				}
			}

			hasFakeFlag := false
			for _, match := range matches {
				flag := string(match)
				// Add the flag if it doesn't already exist
				if !contains(flow.Flags, flag) {
					flow.Flags = append(flow.Flags, flag)
				}
				// Check if it is a fake flag
				if !hasFakeFlag && !flagValidator.IsValid(flag, flowItem.Time) {
					tags = append(tags, "fake-flag")
					hasFakeFlag = true
				}
			}

			for _, tag := range tags {
				// Add the tag if it doesn't already exist
				if !contains(flow.Tags, tag) {
					flow.Tags = append(flow.Tags, tag)
				}
			}
		}
	}

	// Different repr may have multiple duplicate flags between each other, so assume that the "max" inside a repr is the most accurate value
	flow.Flags_In += flagsIn
	flow.Flags_Out += flagsOut
}

// Apply flagids to the entire flow.
// This assumes the `Data` part of the flowItem is already pre-processed, s.t.
func ApplyFlagids(flow *db.FlowEntry, flagidsDb []db.FlagId) {

	var flagids []string
	var matches = make(map[int]int)

	for _, flagid := range flagidsDb {
		flagids = append(flagids, flagid.Content)
	}

	matcher := ahocorasick.NewStringMatcher(flagids)
	for idx := 0; idx < len(flow.Flow); idx++ {
		flowItem := &flow.Flow[idx]
		found := matcher.Match([]byte(flowItem.Data))

		if len(found) > 0 {
			var tag string

			if flowItem.From == "c" {
				tag = "flagid-in"
			} else {
				tag = "flagid-out"
			}

			// Add the tag if it doesn't already exist
			if !contains(flow.Tags, tag) {
				flow.Tags = append(flow.Tags, tag)
			}

			for _, match := range found {
				matches[match] = 1
			}
		}
	}

	for match, _ := range matches {
		flow.Flagids = append(flow.Flagids, flagids[match])
	}
}
