package main

import (
	"encoding/base64"
	"encoding/binary"
	"log"
	"strconv"
	"time"
)

type FlagValidator interface {
	IsValid(flag string, refTime time.Time) bool
}

type DummyFlagValidator struct{}

func (f *DummyFlagValidator) IsValid(flag string, refTime time.Time) bool { return true }

// Helper function for time validation
func IsFlagTimeValid(timeFromFlag, referenceTime time.Time, tolerance time.Duration) bool {
	return timeFromFlag.Before(referenceTime.Add(tolerance)) && timeFromFlag.After(referenceTime.Add(-tolerance))
}


// Team net checking can be disabled by setting teamNet to -1.
// Time checking can be disabled by setting timeTolerance to zero.
type FaustFlagValidator struct {
	teamNet       int
	timeTolerance time.Duration
	xorString     string
}

func (validator *FaustFlagValidator) IsValid(flag string, refTime time.Time) bool {
	const RAW_FLAG_DATA_LEN = 32
	const FLAG_DATA_LEN = 8 + 4 + 2
	data, err := base64.StdEncoding.DecodeString(flag[len(flag)-RAW_FLAG_DATA_LEN:])
	if err != nil {
		// We weren't able to decode it, probably fake flag
		log.Printf("Error during decode of flag %q: %s\n", flag, err)
		return false
	}
	if len(data) < FLAG_DATA_LEN {
		return false
	}

	for x := range [FLAG_DATA_LEN]int{} {
		data[x] = data[x] ^ validator.xorString[x]
	}

	flagTime := time.UnixMilli(int64(binary.BigEndian.Uint64(data[:8])))
	// flagId := int(binary.BigEndian.Uint32(data[8:12]))
	teamNet := int(binary.BigEndian.Uint16(data[12:14]))

	return (validator.teamNet == -1 || validator.teamNet == teamNet) &&
		(validator.timeTolerance == 0 || IsFlagTimeValid(flagTime, refTime, validator.timeTolerance))
}

// Team ID checking can be disabled by setting teamId to -1.
// Time checking can be disabled by setting timeTolerance, startTime and/or tickLength to zero.
type EnowarsFlagValidator struct {
	teamId        int
	serviceCount  int
	maxFlagStores int
	timeTolerance time.Duration
	startTime     time.Time
	tickLength    time.Duration
}

func (validator *EnowarsFlagValidator) IsValid(flag string, refTime time.Time) bool {
	const RAW_FLAG_DATA_LEN = 48
	const FLAG_DATA_LEN = 4 * 4
	data, err := base64.StdEncoding.DecodeString(flag[len(flag)-RAW_FLAG_DATA_LEN:])
	if err != nil {
		// We weren't able to decode it, probably fake flag
		log.Printf("Error during decode of flag %q: %s\n", flag, err)
		return false
	}
	if len(data) < FLAG_DATA_LEN {
		return false
	}

	serviceId := int(binary.LittleEndian.Uint32(data[0:4])) // = Service
	roundOffset := int(binary.LittleEndian.Uint32(data[4:8])) // Flag store
	ownerId := int(binary.LittleEndian.Uint32(data[8:12])) // = Team
	roundId := binary.LittleEndian.Uint32(data[12:16]) // = Tick

	return (validator.teamId == -1 || validator.teamId == ownerId) &&
		serviceId <= validator.serviceCount &&
		roundOffset <= validator.maxFlagStores &&
		(validator.startTime.IsZero() ||
			validator.tickLength <= 0 ||
			validator.timeTolerance == 0 ||
			IsFlagTimeValid(validator.startTime.Add(time.Duration(roundId) * validator.tickLength), refTime, validator.timeTolerance))
}

// Team ID checking can be disabled by setting teamId to -1.
// Time checking can be disabled by setting timeTolerance, startTime and/or tickLength to zero.
type ItallyADFlagValidator struct {
	teamId        int
	serviceCount  int
	timeTolerance time.Duration
	startTime     time.Time
	tickLength    time.Duration
}

func (validator *ItallyADFlagValidator) IsValid(flag string, refTime time.Time) bool {
	var round, team, service int64
	var err error

	round, err = strconv.ParseInt(flag[0:2], 36, 0) // = Tick
	if err != nil {
		return false
	}
	team, err = strconv.ParseInt(flag[3:4], 36, 0) // = Team
	if err != nil {
		return false
	}
	service, err = strconv.ParseInt(flag[5:6], 36, 0) // = Service
	if err != nil {
		return false
	}

	return (validator.teamId == -1 || validator.teamId == int(team)) &&
		int(service) <= validator.serviceCount &&
		(validator.startTime.IsZero() ||
			validator.tickLength <= 0 ||
			validator.timeTolerance == 0 ||
			IsFlagTimeValid(validator.startTime.Add(time.Duration(round) * validator.tickLength), refTime, validator.timeTolerance))
}
