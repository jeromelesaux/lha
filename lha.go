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
	}
}
