package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
	"unsafe"
)

var (
	getPtr      int
	putPtr      int
	storageSize int
	startPtr    int
	ptr         *[]byte
	convertCase bool
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

func getByte() byte {
	v := (*ptr)[getPtr] & 0xff
	getPtr++
	return v
}

func skipBytes(len int) {
	getPtr += len
}

func dumpGetByte() byte {
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
	b0 = int(getByte())
	b1 = int(getByte())
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
	var l int
	b0 := getByte()
	b1 := getByte()
	b2 := getByte()
	b3 := getByte()

	l = int((b3 << 24) + (b2 << 16) + (b1 << 8) + b0)
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

func genericToUnixStamp(t int64) int64 {
	return time.Unix(t, 0).Unix()
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

func (l *LzHeader) getExtendedHeader(fp *io.Reader, headerSize int, hcrc *uint) (error, int) {
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
		nb, err := (*fp).Read(data[:l.headerSize])
		if err != nil || nb == 0 {
			return fmt.Errorf("Invalid header (LHA file ?)"), 0
		}
		extType = int(getByte())
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
		if *hcrc != 0 {
			*hcrc = calcCrc(*hcrc, (*[]byte)(unsafe.Pointer(&data)), uint(getPtr), uint(headerSize))
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

const (
	I_HEADER_SIZE     = 0  /* level 0,1,2   */
	I_HEADER_CHECKSUM = 1  /* level 0,1     */
	I_METHOD          = 2  /* level 0,1,2,3 */
	I_PACKED_SIZE     = 7  /* level 0,1,2,3 */
	I_ATTRIBUTE       = 19 /* level 0,1,2,3 */
	I_HEADER_LEVEL    = 20 /* level 0,1,2,3 */

	COMMON_HEADER_SIZE = 21 /* size of common part */

	I_GENERIC_HEADER_SIZE = 24 /* + name_length */
	I_LEVEL0_HEADER_SIZE  = 36 /* + name_length (unix extended) */
	I_LEVEL1_HEADER_SIZE  = 27 /* + name_length */
	I_LEVEL2_HEADER_SIZE  = 26 /* + padding */
	I_LEVEL3_HEADER_SIZE  = 32
)

var (
	defaultSystemKanjiCode   = none
	optionalArchiveKanjiCode = none
	optionalSystemKanjiCode  = none
	optionalArchiveDelim     = ""
	optionalSystemDelim      = ""
	optionalFilenameCase     = none
)

/*
 * level 0 header
 *
 *
 * offset  size  field name
 * ----------------------------------
 *     0      1  header size    [*1]
 *     1      1  header sum
 *            ---------------------------------------
 *     2      5  method ID                         ^
 *     7      4  packed size    [*2]               |
 *    11      4  original size                     |
 *    15      2  time                              |
 *    17      2  date                              |
 *    19      1  attribute                         | [*1] header size (X+Y+22)
 *    20      1  level (0x00 fixed)                |
 *    21      1  name length                       |
 *    22      X  pathname                          |
 * X +22      2  file crc (CRC-16)                 |
 * X +24      Y  ext-header(old style)             v
 * -------------------------------------------------
 * X+Y+24        data                              ^
 *                 :                               | [*2] packed size
 *                 :                               v
 * -------------------------------------------------
 *
 * ext-header(old style)
 *     0      1  ext-type ('U')
 *     1      1  minor version
 *     2      4  UNIX time
 *     6      2  mode
 *     8      2  uid
 *    10      2  gid
 *
 * attribute (MS-DOS)
 *    bit1  read only
 *    bit2  hidden
 *    bit3  system
 *    bit4  volume label
 *    bit5  directory
 *    bit6  archive bit (need to backup)
 *
 */
func (l *LzHeader) getHeaderLevel0(fp *io.Reader, data []byte) (error, bool) {
	var headerSize int
	var remainSize int
	var extendSize int
	var checksum int
	var nameLength int
	var i int

	l.sizeFieldLength = 2 /* in bytes */
	l.headerSize = int(getByte())
	headerSize = l.headerSize
	checksum = int(getByte())

	/* The data variable has been already read as COMMON_HEADER_SIZE bytes.
	So we must read the remaining header size by the header_size. */
	remainSize = headerSize + 2 - COMMON_HEADER_SIZE
	if remainSize <= 0 {
		return fmt.Errorf("Invalid header size (LHarc file ?)"), false

	}
	nb, err := (*fp).Read(data[COMMON_HEADER_SIZE : COMMON_HEADER_SIZE+remainSize])

	if err != nil || nb == 0 {
		return fmt.Errorf("Invalid header (LHarc file ?)"), false
	}

	if calcSum((*[]byte)(unsafe.Pointer(&data)), I_METHOD, headerSize) != checksum {
		return fmt.Errorf("Checksum error (LHarc file?)"), false
	}

	getBytes((*[]byte)(unsafe.Pointer(&l.method)), 5, 2) // sizeof
	l.packedSize = getLongword()
	l.originalSize = getLongword()
	l.unixLastModifiedStamp = genericToUnixStamp(int64(getLongword()))
	l.attribute = getByte() /* MS-DOS attribute */
	l.headerLevel = getByte()
	nameLength = int(getByte())
	i = getBytes((*[]byte)(unsafe.Pointer(&l.name)), nameLength, 2-1) // sizeof l.name
	l.name[i] = 0

	/* defaults for other type */
	l.unixMode = uint16(unixFileRegular) | uint16(unixRwRwRw)
	l.unixGid = 0
	l.unixUID = 0

	extendSize = headerSize + 2 - nameLength - 24

	if extendSize < 0 {
		if extendSize == -2 {
			/* CRC field is not given */
			l.extendType = extendGeneric
			l.hasCrc = false

			return nil, true
		}

		return fmt.Errorf("Unkonwn header (lha file?)"), false
	}

	l.hasCrc = true
	l.crc = uint(getWord())

	if extendSize == 0 {
		return nil, true
	}

	l.extendType = getByte()
	extendSize--

	if l.extendType == extendUnix {
		if extendSize >= 11 {
			l.minorVersion = getByte()
			l.unixLastModifiedStamp = int64(getLongword())
			l.unixMode = uint16(getWord())
			l.unixUID = uint16(getWord())
			l.unixGid = uint16(getWord())
			extendSize -= 11
		} else {
			l.extendType = extendGeneric
		}
	}
	if extendSize > 0 {
		skipBytes(extendSize)
	}

	l.headerSize += 2
	return nil, true
}

/*
 * level 1 header
 *
 *
 * offset   size  field name
 * -----------------------------------
 *     0       1  header size   [*1]
 *     1       1  header sum
 *             -------------------------------------
 *     2       5  method ID                        ^
 *     7       4  skip size     [*2]               |
 *    11       4  original size                    |
 *    15       2  time                             |
 *    17       2  date                             |
 *    19       1  attribute (0x20 fixed)           | [*1] header size (X+Y+25)
 *    20       1  level (0x01 fixed)               |
 *    21       1  name length                      |
 *    22       X  filename                         |
 * X+ 22       2  file crc (CRC-16)                |
 * X+ 24       1  OS ID                            |
 * X +25       Y  ???                              |
 * X+Y+25      2  next-header size                 v
 * -------------------------------------------------
 * X+Y+27      Z  ext-header                       ^
 *                 :                               |
 * -----------------------------------             | [*2] skip size
 * X+Y+Z+27       data                             |
 *                 :                               v
 * -------------------------------------------------
 *
 */
func (l *LzHeader) getHeaderLevel1(fp *io.Reader, data []byte) (err error, ok bool) {
	var headerSize int
	var remainSize int
	var extendSize int
	var checksum int
	var nameLength int
	var i, dummy int

	l.sizeFieldLength = 2 /* in bytes */
	l.headerSize = int(getByte())
	headerSize = l.headerSize
	checksum = int(getByte())

	/* The data variable has been already read as COMMON_HEADER_SIZE bytes.
	So we must read the remaining header size by the header_size. */
	remainSize = headerSize + 2 - COMMON_HEADER_SIZE
	if remainSize <= 0 {
		return fmt.Errorf("Invalid header size (LHarc file ?)"), false
	}
	nb, err := (*fp).Read(data[COMMON_HEADER_SIZE : COMMON_HEADER_SIZE+remainSize])
	if err != nil || nb == 0 {
		return fmt.Errorf("Invalid header (LHarc file ?)"), false
	}

	if calcSum((*[]byte)(unsafe.Pointer(&data)), I_METHOD, headerSize) != checksum {
		return fmt.Errorf("Checksum error (LHarc file?)"), false
	}

	getBytes((*[]byte)(unsafe.Pointer(&l.method)), 5, 2) //sizeof(hdr->method)
	l.packedSize = getLongword()                         /* skip size */
	l.originalSize = getLongword()
	l.unixLastModifiedStamp = genericToUnixStamp(int64(getLongword()))
	l.attribute = getByte() /* 0x20 fixed */
	l.headerLevel = getByte()

	nameLength = int(getByte())
	i = getBytes((*[]byte)(unsafe.Pointer(&l.name)), nameLength, 2-1) // sizeof(hdr->name)
	l.name[i] = 0

	/* defaults for other type */
	l.unixMode = uint16(unixFileRegular) | uint16(unixRwRwRw)
	l.unixGid = 0
	l.unixUID = 0

	l.hasCrc = true
	l.crc = uint(getWord())
	l.extendType = getByte()

	dummy = headerSize + 2 - nameLength - I_LEVEL1_HEADER_SIZE
	if dummy > 0 {
		skipBytes(dummy) /* skip old style extend header */
	}

	extendSize = getWord()
	var trash uint
	err, extendSize = l.getExtendedHeader(fp, extendSize, &trash)
	if err != nil || extendSize == -1 {
		return err, false
	}

	/* On level 1 header, size fields should be adjusted. */
	/* the `packed_size' field contains the extended header size. */
	/* the `header_size' field does not. */
	l.packedSize -= extendSize
	l.headerSize += extendSize + 2

	return nil, true
}

/*
 * level 2 header
 *
 *
 * offset   size  field name
 * --------------------------------------------------
 *     0       2  total header size [*1]           ^
 *             -----------------------             |
 *     2       5  method ID                        |
 *     7       4  packed size       [*2]           |
 *    11       4  original size                    |
 *    15       4  time                             |
 *    19       1  RESERVED (0x20 fixed)            | [*1] total header size
 *    20       1  level (0x02 fixed)               |      (X+26+(1))
 *    21       2  file crc (CRC-16)                |
 *    23       1  OS ID                            |
 *    24       2  next-header size                 |
 * -----------------------------------             |
 *    26       X  ext-header                       |
 *                 :                               |
 * -----------------------------------             |
 * X +26      (1) padding                          v
 * -------------------------------------------------
 * X +26+(1)      data                             ^
 *                 :                               | [*2] packed size
 *                 :                               v
 * -------------------------------------------------
 *
 */
func (l *LzHeader) getHeaderLevel2(fp *io.Reader, data []byte) (error, bool) {
	var headerSize int
	var remainSize int
	var extendSize int
	var padding int
	var hcrc uint

	l.sizeFieldLength = 2 /* in bytes */
	l.headerSize = getWord()
	headerSize = l.headerSize

	/* The data variable has been already read as COMMON_HEADER_SIZE bytes.
	So we must read the remaining header size without ext-header. */
	remainSize = headerSize - I_LEVEL2_HEADER_SIZE
	if remainSize < 0 {
		return fmt.Errorf("Invalid header size (LHarc file ?)"), false
	}
	n, err := (*fp).Read(data[COMMON_HEADER_SIZE : COMMON_HEADER_SIZE+(I_LEVEL2_HEADER_SIZE-COMMON_HEADER_SIZE)])
	if err != nil || n == 0 {
		return fmt.Errorf("Invalid header (LHarc file ?)"), false
	}

	getBytes((*[]byte)(unsafe.Pointer(&l.method)), 5, 2) // sizeof(hdr->method)
	l.packedSize = getLongword()
	l.originalSize = getLongword()
	l.unixLastModifiedStamp = int64(getLongword())
	l.attribute = getByte() /* reserved */
	l.headerLevel = getByte()

	/* defaults for other type */
	l.unixMode = uint16(unixFileRegular) | uint16(unixRwRwRw)
	l.unixGid = 0
	l.unixUID = 0

	l.hasCrc = true
	l.crc = uint(getWord())
	l.extendType = getByte()
	extendSize = getWord()

	initialize_crc(&hcrc)

	hcrc = calcCrc(hcrc, (*[]byte)(unsafe.Pointer(&data)), 0, uint(n-len(data)))
	err, extendSize = l.getExtendedHeader(fp, extendSize, &hcrc)
	if err != nil || extendSize == -1 {
		return err, false
	}

	padding = headerSize - I_LEVEL2_HEADER_SIZE - extendSize
	/* padding should be 0 or 1 */
	if padding != 0 && padding != 1 {
		return fmt.Errorf("Invalid header size (padding: %d)", padding), false
	}
	padding--
	for padding != 0 {
		buf := make([]byte, 1)
		_, err := (*fp).Read(buf)
		if err != nil {
			return err, false
		}
		hcrc = updateCrc(&hcrc, uint(buf[0]))
	}

	if l.headerCrc != hcrc {
		return fmt.Errorf("header CRC error"), false
	}

	return nil, true
}

/*
 * level 3 header
 *
 *
 * offset   size  field name
 * --------------------------------------------------
 *     0       2  size field length (4 fixed)      ^
 *     2       5  method ID                        |
 *     7       4  packed size       [*2]           |
 *    11       4  original size                    |
 *    15       4  time                             |
 *    19       1  RESERVED (0x20 fixed)            | [*1] total header size
 *    20       1  level (0x03 fixed)               |      (X+32)
 *    21       2  file crc (CRC-16)                |
 *    23       1  OS ID                            |
 *    24       4  total header size [*1]           |
 *    28       4  next-header size                 |
 * -----------------------------------             |
 *    32       X  ext-header                       |
 *                 :                               v
 * -------------------------------------------------
 * X +32          data                             ^
 *                 :                               | [*2] packed size
 *                 :                               v
 * -------------------------------------------------
 *
 */
func (l *LzHeader) getHeaderLevel3(fp *io.Reader, data []byte) (error, bool) {
	var headerSize int
	var remainSize int
	var extendSize int
	var padding int
	var hcrc uint

	l.sizeFieldLength = getWord()
	nb, err := (*fp).Read(data[COMMON_HEADER_SIZE : COMMON_HEADER_SIZE+I_LEVEL3_HEADER_SIZE-COMMON_HEADER_SIZE])
	if err != nil || nb == 0 {
		return fmt.Errorf("Invalid header (LHarc file ?)"), false
	}

	getBytes((*[]byte)(unsafe.Pointer(&l.method)), 5, 2) //sizeof(hdr->method)
	l.packedSize = getLongword()
	l.originalSize = getLongword()
	l.unixLastModifiedStamp = int64(getLongword())
	l.attribute = getByte() /* reserved */
	l.headerLevel = getByte()

	/* defaults for other type */
	l.unixMode = uint16(unixFileRegular) | uint16(unixRwRwRw)
	l.unixGid = 0
	l.unixUID = 0

	l.hasCrc = true
	l.crc = uint(getWord())
	l.extendType = getByte()
	l.headerSize = getLongword()
	headerSize = l.headerSize
	remainSize = headerSize - I_LEVEL3_HEADER_SIZE
	if remainSize < 0 {
		return fmt.Errorf("Invalid header size (LHarc file ?)"), false
	}
	extendSize = getLongword()

	initialize_crc(&hcrc)
	hcrc = calcCrc(hcrc, (*[]byte)(unsafe.Pointer(&data)), 0, uint(nb-len(data)))

	err, extendSize = l.getExtendedHeader(fp, extendSize, &hcrc)
	if err != nil || extendSize == -1 {
		return err, false
	}

	padding = remainSize - extendSize
	/* padding should be 0 */
	if padding != 0 {
		return fmt.Errorf("Invalid header size (padding: %d)", padding), false
	}

	if l.headerCrc != hcrc {
		return fmt.Errorf("header CRC error"), false
	}

	return nil, true
}

func (l *LzHeader) getHeader(fp *io.Reader) (error, bool) {
	var data [lzheaderStorage]byte

	archiveKanjiCode := codeSJIS
	systemKanjiCode := defaultSystemKanjiCode
	var archiveDelim string = "\377\\" /* `\' is for level 0 header and
	   broken archive. */
	var systemDelim string = "//"
	var filenameCase int = none
	var endMark byte

	setupGet((*[]byte)(unsafe.Pointer(&data)), 0, len(data))
	buf := make([]byte, 1)
	nb, err := (*fp).Read(buf)

	endMark = buf[0]
	if err != nil || endMark == 0 {
		return err, false /* finish */
	}
	data[0] = endMark

	nb, err = (*fp).Read(data[1:COMMON_HEADER_SIZE])
	if err != nil || nb == 0 {
		return fmt.Errorf("Invalid header (LHarc file ?)"), false
	}

	switch data[I_HEADER_LEVEL] {
	case 0:
		err, header := l.getHeaderLevel0(fp, data[:])
		if err != nil || header == false {
			return err, false
		}
	case 1:
		err, header := l.getHeaderLevel1(fp, data[:])
		if err != nil || header == false {
			return err, false
		}
	case 2:
		err, header := l.getHeaderLevel2(fp, data[:])
		if err != nil || header == false {
			return err, false
		}
	case 3:
		err, header := l.getHeaderLevel3(fp, data[:])
		if err != nil || header == false {
			return err, false
		}
	default:
		return fmt.Errorf("Unknown level header (level %d)", data[I_HEADER_LEVEL]), false
	}

	/* filename conversion */
	switch l.extendType {
	case extendMsdos:
		filenameCase = none
		if convertCase {
			filenameCase = toLower
		}
	case extendHuman:
	case extendOs68k:
	case extendXosk:
	case extendUnix:
	case extendJava:
		filenameCase = none

	case extendMacos:
		archiveDelim = "\377/:\\"
		/* `\' is for level 0 header and broken archive. */
		systemDelim = "/://"
		filenameCase = none

	case extendAmiga:
		{
			/* workaround */
			lenName := len(l.name)
			if lenName > 0 && l.name[lenName-1] == lhaPathsep && string(l.method[:]) == lzhuff0Method {
				/* replace with "-lhd-" */
				copy(l.method[:], lzhdirsMethod)
			}
		}
	default:
		filenameCase = none
		if convertCase {
			filenameCase = toLower
		}
	}

	if optionalArchiveKanjiCode != none {
		archiveKanjiCode = optionalArchiveKanjiCode
	}
	if optionalSystemKanjiCode != none {
		systemKanjiCode = optionalSystemKanjiCode
	}
	if optionalArchiveDelim != "" {
		archiveDelim = optionalArchiveDelim
	}
	if optionalSystemDelim != "" {
		systemDelim = optionalSystemDelim
	}
	if optionalFilenameCase != none {
		filenameCase = optionalFilenameCase
	}

	/* kanji code and delimiter conversion */
	convertFilename((*[]byte)(unsafe.Pointer(&l.name)), len(l.name), 1, archiveKanjiCode, systemKanjiCode, archiveDelim, systemDelim, filenameCase)

	if l.unixMode&uint16(unixFileSymlink) == uint16(unixFileSymlink) {

		/* split symbolic link */
		p := strings.Index(string(l.name[:]), "|")

		if p != -1 {
			/* hdr->name is symbolic link name */
			/* hdr->realname is real name */
			copy(l.realname[:], l.name[p:len(l.name)-1])
			/* ok */
		} else {
			return fmt.Errorf("unknown symlink name \"%s\"", l.name), false
		}
	}

	return nil, true
}

func convertFilename(name *[]byte, len, size int, fromCode, toCode int, fromDelim, toDelim string, caseTo int) {
	var i int
	key := make([]byte, 1)
	if fromCode == codeSJIS && caseTo == toLower {
		for i = 0; i < len; i++ {
			key[0] = (*name)[i]
			r, _ := utf8.DecodeRune(key)
			if unicode.IsLower(r) {
				caseTo = none
				break
			}
		}
	}
	for i = 0; i < len; i++ {
		index := strings.Index(fromDelim, string((*name)[i]))
		if index != -1 {
			//s := len(fromDelim)
			(*name)[i] = toDelim[index] //name[i] = to_delim[ptr - from_delim];
			continue
		}
		key[0] = (*name)[i]
		r, _ := utf8.DecodeRune(key)
		if caseTo == toUpper && unicode.IsLower(r) {
			(*name)[i] = byte(unicode.ToUpper(r))
			continue
		}
		if caseTo == toLower && unicode.IsUpper(r) {
			(*name)[i] = byte(unicode.ToLower(r))
			continue
		}
	}

}
