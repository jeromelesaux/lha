package main

import (
	"fmt"
	"os"
	"time"
	"unsafe"
)

var (
	getPtr      int
	putPtr      int
	storageSize int
	startPtr    int
	ptr         *[]byte
)

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
	unixLastModifiedStamp int64
	unixMode              uint16
	unixUID               uint16
	unixGid               uint16
	user                  [256]byte
	group                 [256]byte
}

func NewLzHeader() *LzHeader {
	return &LzHeader{}
}

func (l *LzHeader) InitHeader(fileSize int, headerLevel byte) {
	//var len int
	n := copy(l.method[:], []byte(lzhuff0Method))
	if n != methodTypeStorage {
		fmt.Fprintf(os.Stderr, "copy lzh method differs expected [%d] and [%d] bytes copied\n", methodTypeStorage, n)
	}
	l.packedSize = 0
	l.originalSize = fileSize
	l.attribute = genericAttribute
	l.headerLevel = headerLevel
}

func calcSum(p *[]byte, start, len int) int {
	sum := 0
	for len != 0 {
		sum += int((*p)[start])
		len--
		start++
	}

	return sum & 0xff
}

func getByte() int {
	v := (*ptr)[getPtr] & 0xff
	getPtr++
	return int(v)
}

func skipBytes(len int) {
	getPtr += len
}

func dumpGetByte() int {
	return getByte()
}

func dumpSkipBytes(len int) {
	if len == 0 {
		return
	} else {
		skipBytes(len)
	}
	return
}

func getWord() int {
	var b0, b1, w int
	b0 = getByte()
	b1 = getByte()
	w = (b1 << 8) + b0
	return w
}

func putByte(c byte) {
	(*ptr)[putPtr] = c
	putPtr++
}

func putWord(v int) {
	putByte(byte(v))
	putByte(byte(v >> 8))
}

func getLongword() int {
	b0 := getByte()
	b1 := getByte()
	b2 := getByte()
	b3 := getByte()

	l := (b3 << 24) + (b2 << 16) + (b1 << 8) + b0
	return l
}

func putLongWord(v int) {
	putByte(byte(v))
	putByte(byte(v >> 8))
	putByte(byte(v >> 16))
	putByte(byte(v >> 24))
}

func getBytes(buf *[]byte, len, size int) int {
	var i int
	for i = 0; i < len && i < size; i++ {
		(*buf)[i] = (*ptr)[i]
	}
	getPtr += len
	return i
}

func putBytes(buf []byte, len int) {
	for i := 0; i < len; i++ {
		putByte(buf[i])
	}
}

func subbits(n, off, len int) int {
	return (((n) >> (off)) & ((1 << (len)) - 1))
}

func genericToUnixStamp(t int64) time.Time {
	return time.Unix(t, 0)
}

func setupGet(p *[]byte, index, size int) {
	ptr = p
	getPtr = index
	startPtr = index
	storageSize = size
}

func wintimeToUnixStamp() uint64 {
	var wintime [8]uint64
	epoch := [8]uint64{0x01, 0x9d, 0xb1, 0xde, 0xd5, 0x3e, 0x80, 0x00}
	/* 1970-01-01 00:00:00 (UTC) */
	/* wintime -= epoch */
	var borrow uint64 = 0
	for i := 7; i >= 0; i-- {
		wintime[i] = uint64(getByte()) - epoch[i] - borrow
		if wintime[i] > 0xff {
			borrow = 1
		} else {
			borrow = 0
		}
		wintime[i] &= 0xff
	}
	/* q = wintime / 10000000 */
	var x uint64 = 10000000 /* x: 24bit */
	var q, t uint64
	for i := 0; i < 8; i++ {
		t = (t << 8) + wintime[i] /* 24bit + 8bit. t must be 32bit variable */
		q <<= 8                   /* q must be 32bit (time_t) */
		q += t / x
		t %= x /* 24bit */

	}
	return q
}

/*
 * extended header
 *
 *             size  field name
 *  --------------------------------
 *  base header:         :
 *           2 or 4  next-header size  [*1]
 *  --------------------------------------
 *  ext header:   1  ext-type            ^
 *                ?  contents            | [*1] next-header size
 *           2 or 4  next-header size    v
 *  --------------------------------------
 *
 *  on level 1, 2 header:
 *    size field is 2 bytes
 *  on level 3 header:
 *    size field is 4 bytes
 */

func (l *LzHeader) getExtendedHeader(fp []byte, headerSize int, hcrc uint) (error, int) {
	var data [lzheaderStorage]byte
	var nameLength int
	var dirname [filenameLength]byte
	var dirLength, i int
	wholeSize := headerSize
	var extType int
	n := i + l.sizeFieldLength /* `ext-type' + `next-header size' */
	if l.headerLevel == 0 {
		return nil, 0
	}
	nameLength = len(l.name)
	for l.headerSize != 0 {
		setupGet((*[]byte)(unsafe.Pointer(&data)), 0, len(data))
		if len(data) < l.headerSize {
			return fmt.Errorf("header size (%d) too large.", l.headerSize), 0
		}
		if nb := copy(data[:], fp[:l.headerSize]); nb == 0 {
			return fmt.Errorf("Invalid header (LHA file ?)"), 0
		}
		extType = getByte()
		switch extType {
		case 0:
			l.headerCrc = uint(getWord())
			/* header crc (CRC-16) */
			/* clear buffer for CRC calculation. */
			data[1] = 0
			data[2] = 0
			skipBytes(l.headerSize - n - 2)
		case 1:
			/* filename */
			nameLength = getBytes((*[]byte)(unsafe.Pointer(&l.name)), l.headerSize-n, len(l.name)-1)
			l.name[nameLength] = 0
		case 2:
			dirLength = getBytes((*[]byte)(unsafe.Pointer(&dirname)), headerSize-n, len(dirname)-1)
			dirname[dirLength] = 0
		case 0x40:
			/* MS-DOS attribute */
			l.attribute = byte(getWord())
		case 0x41:
			/* Windows time stamp (FILETIME structure) */
			/* it is time in 100 nano seconds since 1601-01-01 00:00:00 */

			skipBytes(8) /* create time is ignored */

			/* set last modified time */
			if l.headerLevel >= 2 {
				skipBytes(8) /* time_t has been already set */
			} else {
				l.unixLastModifiedStamp = int64(wintimeToUnixStamp())
			}
			skipBytes(8) /* last access time is ignored */
		case 0x42:
			skipBytes(8)
			skipBytes(8)
		case 0x50:
			/* UNIX permission */
			l.unixMode = uint16(getWord())
		case 0x51:
			/* UNIX group name */
			i = getBytes((*[]byte)(unsafe.Pointer(&l.group)), headerSize-n, len(l.group)-1)
			l.group[i] = 0
		case 0x53:
			/* UNIX user name */
			i = getBytes((*[]byte)(unsafe.Pointer(&l.user)), headerSize-n, len(l.user)-1)
			l.user[i] = 0
		case 0x54:
			/* UNIX last modified time */
			l.unixLastModifiedStamp = int64(getLongword())
		default:
			/* other headers */
			/* 0x39: multi-disk header
			   0x3f: uncompressed comment
			   0x42: 64bit large file size
			   0x48-0x4f(?): reserved for authenticity verification
			   0x7d: encapsulation
			   0x7e: extended attribute - platform information
			   0x7f: extended attribute - permission, owner-id and timestamp
			         (level 3 on OS/2)
			   0xc4: compressed comment (dict size: 4096)
			   0xc5: compressed comment (dict size: 8192)
			   0xc6: compressed comment (dict size: 16384)
			   0xc7: compressed comment (dict size: 32768)
			   0xc8: compressed comment (dict size: 65536)
			   0xd0-0xdf(?): operating systemm specific information
			   0xfc: encapsulation (another opinion)
			   0xfe: extended attribute - platform information(another opinion)
			   0xff: extended attribute - permission, owner-id and timestamp
			         (level 3 on UNLHA32) */
			skipBytes(headerSize - n)
		}
		if hcrc != 0 {
			hcrc = calcCrc(hcrc, (*[]byte)(unsafe.Pointer(&data)), uint(getPtr), uint(headerSize))
		}

		if l.sizeFieldLength == 2 {
			headerSize = getWord()
			wholeSize += headerSize
		} else {
			headerSize = getLongword()
			wholeSize += headerSize
		}
	}

	return nil, wholeSize
}
