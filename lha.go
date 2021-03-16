package lha

import "io"

type StringPool struct {
	used   int
	size   int
	n      int
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
	MakeCrcTable()
	headers = make([]*LzHeader, 0)
	fr, err := openOldArchive(l)
	if err != nil {
		return headers, err
	}
	defer fr.Close()
	var or io.Reader
	or = fr
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
	}
	return headers, nil
}

func (l *Lha) Decompress(header *LzHeader) (err error) {
	l.HeaderLevel = 2
	MakeCrcTable()
	return extractWithHeader(header, l)
}

func (l *Lha) CompressBytes(filename string, body []byte, compressMethod int, headerLevel int) (err error) {
	l.HeaderLevel = headerLevel
	MakeCrcTable()
	l.CompressMethod = compressMethod
	l.CmdFilev = append(l.CmdFilev, filename)
	l.CmdFilec++
	return compressStream(l.archiveName, body, l)
}

func (l *Lha) Compress(fileToAdd string, compressMethod int, headerLevel int) (err error) {
	l.HeaderLevel = headerLevel
	MakeCrcTable()
	l.CompressMethod = compressMethod
	l.CmdFilev = append(l.CmdFilev, fileToAdd)
	l.CmdFilec++
	return CommandAdd(l.archiveName, l)
}
