package cmd

import (
	"math"
	"strconv"
	"strings"
	"time"
	"unicode"

	"code.cloudfoundry.org/bytefmt"

	"github.com/dmolesUC3/cos/internal/suite"

	"fmt"

	"github.com/janeczku/go-spinner"
	"github.com/spf13/cobra"

	"github.com/dmolesUC3/cos/internal/logging"
)

type SuiteFlags struct {
	CosFlags
	SizeMax string
	CountMax int64
	DryRun bool
}

func (f *SuiteFlags) sizeMax() (int64, error) {
	sizeStr := f.SizeMax
	sizeIsNumeric := strings.IndexFunc(sizeStr, unicode.IsLetter) == -1
	if sizeIsNumeric {
		return strconv.ParseInt(sizeStr, 10, 64)
	}

	bytes, err := bytefmt.ToBytes(sizeStr)
	if err == nil && bytes > math.MaxInt64 {
		return 0, fmt.Errorf("specified size %d bytes exceeds maximum %d", bytes, math.MaxInt64)
	}
	return int64(bytes), err
}

func init() {
	f := SuiteFlags{}
	cmd := &cobra.Command{
		Use:   "suite <BUCKET-URL>",
		Short: "run a suite of tests",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSuite(args[0], f)
		},
	}
	cmdFlags := cmd.Flags()
	f.AddTo(cmdFlags)

	// TODO: document these
	sizeMaxDefault := bytefmt.ByteSize(bytefmt.GIGABYTE)
	cmdFlags.StringVarP(&f.SizeMax, "sizeMax", "s", sizeMaxDefault, "max file size to create")
	cmdFlags.Int64VarP(&f.CountMax, "countMax", "c", -1, "max number of files to create, or -1 for no limit")
	cmdFlags.BoolVarP(&f.DryRun, "dryRun", "n", false, "dry run")
	rootCmd.AddCommand(cmd)
}

func runSuite(bucketStr string, f SuiteFlags) error {
	// TODO: figure out some sensible way to log while spinning
	// logger := logging.DefaultLoggerWithLevel(f.LogLevel())
	// logger.Tracef("flags: %v\n", f)
	// logger.Tracef("bucket URL: %v\n", bucketStr)

	target, err := f.Target(bucketStr)
	if err != nil {
		return err
	}

	sizeMax, err := f.sizeMax()
	if err != nil {
		return err
	}

	var countMax uint64
	if f.CountMax < 0 {
		countMax = math.MaxUint64
	} else {
		countMax = uint64(f.CountMax)
	}

	//noinspection GoPrintFunctions
	fmt.Println("Starting test suite…\n")

	startAll := time.Now().UnixNano()
	allTasks := suite.AllTasks(sizeMax, countMax)
	for index, task := range allTasks {
		title := fmt.Sprintf("%d. %v", index+1, task.Title())

		// TODO: spin with emoji so width doesn't change
		sp := spinner.StartNew(title)

		var ok bool
		var detail string

		start := time.Now().UnixNano()
		if f.DryRun {
			// More framerate sync shenanigans
			time.Sleep(time.Duration(len(sp.Charset)) * sp.FrameRate)
			ok = true
		} else {
			ok, detail = task.Invoke(target)
		}
		elapsed := time.Now().UnixNano() - start

		// Lock() / Unlock() around Stop() needed to synchronize cursor movement
		// ..but not always enough (thus the sleep above)
		// TODO: file an issue about this
		sp.Lock()
		sp.Stop()
		sp.Unlock()

		var msgFmt string
		if ok {
			msgFmt = "\u2705 %v: successful (%v)"
		} else {
			msgFmt = "\u274C %v: FAILED (%v)"
		}
		msg := fmt.Sprintf(msgFmt, title, logging.FormatNanos(elapsed))
		fmt.Println(msg)

		if detail != "" && f.LogLevel() > logging.Info {
			fmt.Println(detail)
		}
	}
	elapsedAll := time.Now().UnixNano() - startAll
	fmt.Printf("\n…test complete (%v).\n", logging.FormatNanos(elapsedAll))

	return nil
}
