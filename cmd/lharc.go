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
	addfile            = flag.String("a", "", "Add(or replace) to archive ")
	movefile           = flag.Bool("m", false, "Move to archive ")
	createfile         = flag.String("c", "", "re-Construct new archive ")
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

	timestampArchive bool

	quietMode int

	newArchive                  bool
	updateIfNewer               bool
	deleteAfterAppend           bool
	backupOldArchive            bool
	getFilenameFromStdin        bool
	recoverArchiveWhenInterrupt bool
	recursiveArchiving          bool

	temporaryFD         int
	defaultLzhuffMethod = lha.Lzhuff5MethodNum
)

func main() {
	var globalError error
	flag.Parse()

	initVariable() /* Added N.Watazaki */

	if *archiveNameOption == "" {
		printVersion()
		return
	}

	parseOption()
	sortFiles()
	/* make crc table */
	lha.MakeCrcTable()

	switch cmd {
	case lha.CmdExtract:
		globalError = lha.CommandExtract(*archiveNameOption) // to be tested
	case lha.CmdAdd:
		globalError = lha.CommandAdd(*archiveNameOption) // to implement
	case lha.CmdList:
		globalError = lha.CommandList(*archiveNameOption) // to be tested
	case lha.CmdDelete:
		lha.CommadDelete(*archiveNameOption) // to implemennt
	default:
		fmt.Fprintf(os.Stderr, "option unknown.")
		printVersion()
	}

	if globalError != nil {
		fmt.Fprintf(os.Stderr, "Execution error :%v\n", globalError.Error())
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

	if *createfile != "" {
		newArchive = true
		cmd = lha.CmdAdd
		lha.CmdFilev = append(lha.CmdFilev, *createfile)
		lha.CmdFilec++
		return
	}

	if *addfile != "" {
		cmd = lha.CmdAdd
		lha.CmdFilev = append(lha.CmdFilev, *addfile)
		lha.CmdFilec++
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
		lha.VerboseListing = true

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
		lha.HeaderLevel = 0
	}

	if *compressionmethod != 0 {
		lha.CompressMethod = lha.Lzhuff1MethodNum
		lha.HeaderLevel = 0
		switch *compressionmethod {
		case 5:
			lha.CompressMethod = lha.Lzhuff5MethodNum
		case '6':
			lha.CompressMethod = lha.Lzhuff6MethodNum
			break
		case '7':
			lha.CompressMethod = lha.Lzhuff7MethodNum
		default:
			fmt.Fprintf(os.Stderr, "invalid compression method 'o%d'", *compressionmethod)
			return
		}
	}

	if *excludefiles != "" {
		lha.ExcludeFiles = strings.Split(*excludefiles, ",")
	}

	if *headerlevel != 2 {
		switch *headerlevel {
		case 0:
			lha.HeaderLevel = 0
		case 1:
			lha.HeaderLevel = 1
		}
	}
	return
}

func printVersion() {
	flag.PrintDefaults()
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

	lha.CompressMethod = defaultLzhuffMethod /* defined in config.h */

	lha.HeaderLevel = 2 /* level 2 */
	quietMode = 0

	/* view command flags */
	lha.VerboseListing = false

	/* extract command flags */
	lha.OutputToStdout = false

	/* append command flags */
	lha.NewArchive = false
	lha.UpdateIfNewer = false
	lha.DeleteAfterAppend = false
	lha.GenericFormat = false

	recoverArchiveWhenInterrupt = false
	getFilenameFromStdin = false
	lha.IgnoreDirectory = false
	lha.ExcludeFiles = make([]string, 0)
	lha.VerifyMode = false

	lha.ConvertCase = false

	lha.ExtractDirectory = ""
	temporaryFD = -1

	lha.BackupOldArchive = false

	lha.ExtractBrokenArchive = false
	lha.DecodeMacbinaryContents = false
	lha.SortContents = true
	recursiveArchiving = true
	lha.DumpLzss = false
}

func sortFiles() {
	if lha.SortContents {
		sort.Strings(lha.CmdFilev)
	}
}
