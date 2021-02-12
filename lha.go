package main

import "time"

type StringPool struct {
	used   int
	size   int
	n      int
	buffer []byte
}

type LzHeader struct {
	headerSize      int
	sizeFieldLength int
	method          [methodTypeStorage]byte
	packedSize      int
	originalSize    int
	attribute       byte
	headerLevel     byte
	name            [filenameLength]byte
	realname        [filenameLength]byte /* real name for symbolic link */
	crc             uint                 /* file CRC */
	hasCrc          bool                 /* file CRC */
	headerCrc       uint                 /* header CRC */
	extendType      byte
	minorVersion    byte

	/* extend_type == EXTEND_UNIX  and convert from other type. */
	unixLastModifiedStamp time.Time
	unixMode              uint16
	unixUID               uint16
	unixGid               uint16
	user                  [256]byte
	group                 [256]byte
}

type interfacing struct {
	infile   []byte
	outfile  []byte
	original int
	packed   int
	readSize int
	dicbit   int
	method   int
}
