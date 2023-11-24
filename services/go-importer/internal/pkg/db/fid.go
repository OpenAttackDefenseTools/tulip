package db

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"log"
	"time"

	"github.com/gofrs/uuid/v5"
)

func FidPack(t time.Time, bytes_rand []byte) uuid.UUID {
	bytes_time := make([]byte, 8)
	binary.BigEndian.PutUint64(bytes_time, uint64(t.UnixMicro()))
	_, error := rand.Read(bytes_rand)
	if error != nil {
		log.Fatal(error)
	}

	hex_time := make([]byte, 16)
	hex_rand := make([]byte, 14)
	hex.Encode(hex_time, bytes_time)
	hex.Encode(hex_rand, bytes_rand)
	str_time := string(hex_time)
	str_rand := string(hex_rand)

	uuid, error := uuid.FromString(str_time[0:12] + "8" + str_time[12:15] + "8" + str_time[15:16] + str_rand)
	if error != nil {
		log.Fatal(error)
	}
	return uuid
}

func FidCreate(t time.Time) uuid.UUID {
	bytes_rand := make([]byte, 7)
	_, error := rand.Read(bytes_rand)
	if error != nil {
		log.Fatal(error)
	}

	return FidPack(t, bytes_rand)
}
