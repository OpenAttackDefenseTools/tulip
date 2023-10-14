package main

import (
	"encoding/base64"
	"encoding/binary"
	"log"
	"time"
)

const FLAG_DATA_LEN = 8 + 4 + 2

type FlagValidator interface {
	IsValid(flag *string, tick time.Time) bool
}

type FaustFlagValidator struct {
	teamNet uint16
	timeTolerance time.Duration
	prefixLen uint
	xorString string
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
