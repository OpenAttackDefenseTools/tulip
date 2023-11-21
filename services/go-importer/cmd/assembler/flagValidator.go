package main

import (
	"encoding/base64"
	"encoding/binary"
	"log"
	"time"
)

type FlagValidator interface {
	IsValid(flag string) bool
}

type DummyFlagValidator struct{}

func (f DummyFlagValidator) IsValid(flag string) bool { return true }

type FaustFlagValidator struct {
	prefixLen     int
	teamNet       uint16
	timeTolerance time.Duration
	xorString     string
}

/*
func NewFaustFlagValidator(
	teamNetNo uint16,
	flag1 string,
	flag2 string,
	prefixLen uint,
	timeTolerance time.Duration,
) FaustFlagValidator {
	return FaustFlagValidator{
		-1,
		NULL,
		5,
		"CTF-GAMESERVER"
	}
}
*/

func (f FaustFlagValidator) IsValid(flag string) bool {
	const FLAG_DATA_LEN = 8 + 4 + 2
	data, err := base64.StdEncoding.DecodeString(flag[f.prefixLen:])
	if err != nil {
		log.Println("Error during decode of flag: ", err)
		return false
	}

	if len(data) < FLAG_DATA_LEN {
		return false
	}
	for x := range [FLAG_DATA_LEN]int{} {
		data[x] = data[x] ^ f.xorString[x]
	}
	t := time.UnixMilli(int64(binary.BigEndian.Uint64(data[:8])))
	// flagId := binary.BigEndian.Uint32(data[8:12])
	teamNet := binary.BigEndian.Uint16(data[12:14])
	return t.Before(time.Now().Add(-f.timeTolerance)) && teamNet == f.teamNet
}

type EnowarsFlagValidator struct {
	prefixLen     int
	teamId        uint32
	serviceCount  uint32
	maxFlagStores uint32
	// timeTolerance time.Duration
}

func (f EnowarsFlagValidator) IsValid(flag string) bool {
	const FLAG_DATA_LEN = 4 * 4
	data, err := base64.StdEncoding.DecodeString(flag[f.prefixLen:])
	if err != nil {
		log.Println("Error during decode of flag: ", err)
		return false
	}

	if len(data) < FLAG_DATA_LEN {
		return false
	}

	serviceId := binary.LittleEndian.Uint32(data[0:4]) // = Service
	roundOffset := binary.LittleEndian.Uint32(data[4:8]) // Flag store
	ownerId := binary.LittleEndian.Uint32(data[8:12]) // = Team
	// roundId := binary.LittleEndian.Uint32(data[12:16]) // = Tick

	return serviceId <= f.serviceCount && roundOffset <= f.maxFlagStores && f.teamId == ownerId
}
