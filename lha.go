package lha

import (
	"fmt"
	"io"
	"os"
)

type StringPool struct {
	// nolint: unused
	used int
	// nolint: unused
	size int
	// nolint: unused
	n int
	// nolint: unused
	buffer []byte
}

type interfacing struct {
	infile   io.Reader
	outfile  io.Writer
	original int
	packed   int
	readSize int
	dicbit   int
	method   int
}

type Lha struct {
	remainder                      uint
	ExtractBrokenArchive, DumpLzss bool
	subbitbuf                      byte
	bitcount                       byte
	infile                         io.Reader
	outfile                        io.Writer
	unpackable                     bool
	compsize                       int
	origsize                       int
	bitbuf                         uint16
	skipFlg                        bool /* FALSE..No Skip , TRUE..Skip */
	dirinfo                        *LzHeaderList
	Noexec                         bool
	Force                          bool
	IgnoreDirectory                bool
	OutputToStdout                 bool
	Verbose                        bool
	VerboseListing                 bool
	DecodeMacbinaryContents        bool
	NewArchive                     bool
	UpdateIfNewer                  bool
	BackupOldArchive               bool
	newArchive                     bool
	DeleteAfterAppend              bool
	SortContents                   bool
	RecursiveArchiving             bool
	readingFilename                string
	writingFilename                string
	archiveName                    string
	HeaderLevel                    int
	CompressMethod                 int
	mFlag, flagcnt, matchpos       int
	loc                            uint16
	ExtractDirectory               string

	ExcludeFiles         []string
	CmdFilev             []string
	CmdFilec             int
	newArchiveName       string
	mostRecent           int64
	newArchiveNameBuffer []byte
	temporaryName        string
	backupArchiveName    []byte
}

func NewLha(archivename string) *Lha {
	return &Lha{
		archiveName:          archivename,
		dirinfo:              &LzHeaderList{},
		newArchiveNameBuffer: make([]byte, FilenameLength),
		backupArchiveName:    make([]byte, FilenameLength),
		CompressMethod:       Lzhuff5MethodNum,
	}
}

func (l *Lha) Headers() (headers []*LzHeader, err error) {
	var pos int
	// verify mode here, no extraction
	VerifyMode = true
	defer func() {
		VerifyMode = false
	}()
	MakeCrcTable()
	headers = make([]*LzHeader, 0)
	fr, err := openOldArchive(l)
	if err != nil {
		return headers, err
	}
	defer fr.Close()
	var or io.Reader = fr
	for {
		h := NewLzHeader()
		err, hasHeader := h.GetHeader(&or)
		if !hasHeader {
			break
		}
		if err != nil {
			break
		}
		headers = append(headers, h)
		pos = 0
		// seek io.reader to + file size
		if l.needFile(string(h.Name[:])) {
			readSize, err := l.extractOne(&or, h)
			if err != nil {
				return headers, err
			}
			if readSize != h.PackedSize {
				/* when error occurred in extract_one(), should adjust
				   point of file stream */
				if err := skipToNextpos(&or, pos, h.PackedSize, readSize); err != nil {
					return headers, fmt.Errorf("cannot seek to next header position from \"%s\"", h.Name)
				}
			}
		} else {
			if err := skipToNextpos(&or, pos, h.PackedSize, 0); err == nil {
				return headers, fmt.Errorf("cannot seek to next header position from \"%s\"", h.Name)
			}
		}
	}
	return headers, nil
}

func (l *Lha) Decompress(header *LzHeader) (err error) {
	l.HeaderLevel = 2
	MakeCrcTable()
	return extractWithHeader(header, l)
}

func (l *Lha) DecompresBytes(header *LzHeader) ([]byte, error) {
	l.HeaderLevel = 2
	MakeCrcTable()
	return extractBytesWithHeader(header, l)
}

func (l *Lha) CompressBytes(filename string, body []byte, compressMethod int, headerLevel int) (err error) {
	l.HeaderLevel = headerLevel
	MakeCrcTable()
	l.CompressMethod = compressMethod
	l.CmdFilev = append(l.CmdFilev, filename)
	l.CmdFilec++
	_, err = os.Stat(l.archiveName)
	if err != nil && os.IsNotExist(err) {
		l.newArchive = true
	} else {
		if !os.IsExist(err) {
			return err
		}
	}

	return compressBytes(l.archiveName, body, l)
}

func (l *Lha) Compress(fileToAdd string, compressMethod int, headerLevel int) (err error) {
	_, err = os.Stat(l.archiveName)
	if err != nil && os.IsNotExist(err) {
		l.newArchive = true
	} else {
		if !os.IsExist(err) {
			return err
		}
	}
	l.HeaderLevel = headerLevel
	MakeCrcTable()
	l.CompressMethod = compressMethod
	l.CmdFilev = append(l.CmdFilev, fileToAdd)
	l.CmdFilec++
	return CommandAdd(l.archiveName, l)
}
