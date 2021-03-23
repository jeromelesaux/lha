package lha

import (
	"fmt"
	"io"
)

var (
	getcEucCache int
	TextMode     bool
	VerifyMode   bool
)

const (
	EOF int = 0
)

func initCodeCache() {
	getcEucCache = EOF
}

func calcCrc(crc uint, p *[]byte, pIndex, n uint) uint {
	for n > 0 {
		crc = updateCrc(crc, (*p)[pIndex])
		pIndex++
		n--
	}
	return crc
}

func MakeCrcTable() {
	var i, j, r uint
	for i = 0; i <= ucharMax; i++ {
		r = i
		for j = 0; j < uint(charBit); j++ {
			if r&1 != 0 {
				r = (r >> 1) ^ crcpoly
			} else {
				r >>= 1
			}
		}
		crctable[i] = r
	}
}

func (l *Lha) freadCrc(crcp *uint, p *[]byte, pindex uint, n int, fp io.Reader) (int, error) {
	var err error
	if TextMode {
		n, err := l.freadTxt(p, pindex, n, fp)
		if err != nil {
			return n, err
		}
	} else {
		buf := make([]byte, 0, n)
		n, err = io.ReadFull(fp, buf[:cap(buf)])
		buf = buf[:n]
		//n, err := fp.Read(buf)
		if err != nil {
			if err != io.ErrUnexpectedEOF {
				return n, err
			}
		}
		copy((*p)[pindex:int(pindex)+n], buf[:])
	}

	*crcp = calcCrc(*crcp, p, pindex, uint(n))

	return n, nil
}

func (l *Lha) freadTxt(vp *[]byte, pindex uint, n int, fp io.Reader) (int, error) {
	var c byte
	var cnt int
	var p *[]byte = vp

	for cnt < n {
		if getcEucCache != EOF {
			c = byte(getcEucCache)
			getcEucCache = EOF
		} else {
			var b [1]byte
			_, err := fp.Read(b[:])
			if err != nil {
				return cnt, err
			}
			c = b[0]

			if c == '\n' {
				getcEucCache = int(c)
				l.origsize++
				c = '\r'
			}
		}
		(*p)[pindex] = c
		pindex++
		cnt++
	}
	return cnt, nil
}

func (l *Lha) fwriteCrc(crcp *uint, p []byte, n int, fp *io.Writer) error {
	*crcp = calcCrc(*crcp, &p, 0, uint(n))

	if VerifyMode {
		return nil
	}
	_, err := (*fp).Write(p)
	if err != nil {
		return fmt.Errorf("file write error :%v", err.Error())
	}
	return nil
}
