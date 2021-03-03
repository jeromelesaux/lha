package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/jeromelesaux/lha"
)

const ()

var (
	version            = flag.Bool("version", false, "print version of application ")
	listfile           = flag.Bool("l", false, "List ")
	listfileverbose    = flag.Bool("lv", false, "Verbose List ")
	extractfile        = flag.Bool("e", false, "EXtract from archive ")
	updatefile         = flag.Bool("u", false, "Update newer files to archive ")
	deletefile         = flag.Bool("d", false, "Delete from archive ")
	addfile            = flag.Bool("a", false, "Add(or replace) to archive ")
	movefile           = flag.Bool("m", false, "Move to archive ")
	createfile         = flag.Bool("c", false, "re-Construct new archive ")
	testcrcfile        = flag.Bool("t", false, "Test file CRC in archive ")
	verbosemodeoption  = flag.Bool("v", false, "verbose ")
	nonexecuteoption   = flag.Bool("n", false, "not execute")
	forceoption        = flag.Bool("f", false, "force (over write at extract) ")
	printstdoutarchive = flag.Bool("p", false, "Print to STDOUT from archive ")
	genericformat      = flag.Int("g", 2, " Generic format (for compatibility) 0/1/2 header level (a/u/c) ")
	compressionmethod  = flag.Int("o", 0, "o[567] compression method (a/u/c) ")
	headerlevel        = flag.Int("h", 2, "0/1/2 header level (a/u/c) ")
	excludefiles       = flag.String("x", "", "x=<pattern>  eXclude files (a/u/c), format : -x file1,file2,file3 ")
	archiveNameOption  = flag.String("archive", "", "archive file path ")

	cmd byte = lha.CmdUnknown

	timestampArchive            bool
	compressMethod              int
	headerLevel                 int
	quietMode                   int
	verboseListing              bool
	newArchive                  bool
	updateIfNewer               bool
	deleteAfterAppend           bool
	backupOldArchive            bool
	getFilenameFromStdin        bool
	recoverArchiveWhenInterrupt bool
	recursiveArchiving          bool
	sortContents                bool
	excludeFiles                []string
	temporaryFD                 int
	defaultLzhuffMethod         = lha.Lzhuff5MethodNum
)

func main() {
	var globalError error
	flag.Parse()

	initVariable() /* Added N.Watazaki */

	if *archiveNameOption == "" {
		flag.PrintDefaults()
		printVersion()
	} else {
		lha.ArchiveName = *archiveNameOption
	}

	parseOption()
	sortFiles()
	/* make crc table */
	lha.MakeCrcTable()

	switch cmd {
	case lha.CmdExtract:
		globalError = lha.CommandExtract()
	case lha.CmdAdd:
		lha.CommandAdd()
	case lha.CmdList:
		lha.CommandList()
	case lha.CmdDelete:
		lha.CommadDelete()
	default:
		fmt.Fprintf(os.Stderr, "option unknown.")
	}

	if globalError != nil {
		os.Exit(1)
	}
	os.Exit(0)
}

/*
  Parse LHA command and options.
*/
func parseOption() {
	if *listfile {
		cmd = lha.CmdList
		lha.CmdFilev = make([]string, 0)
		return
	}
	if *extractfile {
		cmd = lha.CmdExtract
		return
	}

	if *printstdoutarchive {
		lha.OutputToStdout = true
		cmd = lha.CmdExtract
		return
	}

	if *createfile {
		newArchive = true
		cmd = lha.CmdAdd
		return
	}

	if *addfile {
		cmd = lha.CmdAdd

	}

	if *deletefile {
		cmd = lha.CmdDelete

	}

	if *updatefile {
		cmd = lha.CmdAdd
		updateIfNewer = true

	}

	if *movefile {
		cmd = lha.CmdAdd
		deleteAfterAppend = true

	}

	if *listfile {
		cmd = lha.CmdList
	}
	if *listfileverbose {
		cmd = lha.CmdList
		verboseListing = true

	}

	if *testcrcfile {
		cmd = lha.CmdExtract
		lha.VerifyMode = true

	}
	parseSubOption()
	return
}

func parseSubOption() {
	if *genericformat != 5 {
		lha.GenericFormat = true
		headerLevel = 0
	}

	if *compressionmethod != 0 {
		compressMethod = lha.Lzhuff1MethodNum
		headerLevel = 0
		switch *compressionmethod {
		case 5:
			compressMethod = lha.Lzhuff5MethodNum
		case '6':
			compressMethod = lha.Lzhuff6MethodNum
			break
		case '7':
			compressMethod = lha.Lzhuff7MethodNum
		default:
			fmt.Fprintf(os.Stderr, "invalid compression method 'o%d'", *compressionmethod)
			return
		}
	}

	if *excludefiles != "" {
		excludeFiles = strings.Split(*excludefiles, ",")
	}

	if *headerlevel != 2 {
		switch *headerlevel {
		case 0:
			headerLevel = 0
		case 1:
			headerLevel = 1
		}
	}
	return
}

func printVersion() {
	fmt.Fprintf(os.Stdout, "%s version %s (%s)\n", lha.PackageName, lha.PackageVersion, lha.PlatForm)
	//fmt.Fprintf(os.Stdout, "  configure options: %s\n", lhaConfigureOptions)
}

func initVariable() { /* Added N.Watazaki */
	/* options */
	lha.Quiet = false
	lha.TextMode = false
	lha.Verbose = false
	lha.Noexec = false /* debugging option */
	lha.Force = false
	timestampArchive = false

	compressMethod = defaultLzhuffMethod /* defined in config.h */

	headerLevel = 2 /* level 2 */
	quietMode = 0

	/* view command flags */
	verboseListing = false

	/* extract command flags */
	lha.OutputToStdout = false

	/* append command flags */
	newArchive = false
	updateIfNewer = false
	deleteAfterAppend = false
	lha.GenericFormat = false

	recoverArchiveWhenInterrupt = false
	getFilenameFromStdin = false
	lha.IgnoreDirectory = false
	excludeFiles = make([]string, 0)
	lha.VerifyMode = false

	lha.ConvertCase = false

	lha.ExtractDirectory = ""
	temporaryFD = -1

	backupOldArchive = false

	lha.ExtractBrokenArchive = false
	lha.DecodeMacbinaryContents = false
	sortContents = true
	recursiveArchiving = true
	lha.DumpLzss = false
}

func sortFiles() {
	if sortContents {
		sort.Strings(lha.CmdFilev)
	}
}
