package cmd

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/heedy/heedy/backend/assets"
	"github.com/heedy/heedy/backend/buildinfo"
	"github.com/heedy/heedy/backend/server"
	"github.com/heedy/heedy/backend/updater"
)

var (

	// ErrTooManyArgs is called when given too many args
	ErrTooManyArgs = errors.New("Too many arguments were specified.")
)

var verbose bool
var revert bool

var force_run bool

// RootCmd is the root command under which all other commands are placed.
// It is used to initialize all variables that are global for the whole app
var RootCmd = &cobra.Command{
	Use:     "heedy",
	Short:   "Heedy is an IoT and QS repository and analysis system",
	Long:    `Heedy is a database built for interacting with your IoT devices and for storing your quantified-self data. It is buit to be extensible and self-contained, with a powerful plugin system allowing for in-depth analysis and action.`,
	Version: buildinfo.Version,
	RunE: func(cmd *cobra.Command, args []string) error {
		c := assets.NewConfiguration()
		c.Verbose = verbose

		// Check if a database exists in the root directory. If it doesn't runs the equivalent of "heedy create"
		directory, err := os.UserConfigDir()
		if err != nil {
			return err
		}
		directory = path.Join(directory, "heedy")
		if _, err := os.Stat(path.Join(directory, "heedy.conf")); os.IsNotExist(err) {
			// A heedy database does not exist in the config directory

			return server.Setup(server.SetupContext{
				Config:    c,
				Directory: directory,
			}, ":1324")
		}

		directory, err = filepath.Abs(directory)
		if err != nil {
			return err
		}
		logrus.Infof("Using database at %s", directory)
		if err = writepid(directory); err != nil {
			return err
		}

		return updater.Run(updater.Options{
			ConfigDir:   directory,
			AddonConfig: c,
			Revert:      revert,
			Runner: func(a *assets.Assets) error {
				return server.Run(a, nil)
			},
		})
	},
}

// Execute runs the root command
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func GetDirectory(args []string) (string, error) {
	if len(args) > 1 {
		return "", ErrTooManyArgs
	}
	var directory string
	if len(args) == 1 {
		directory = args[0]
	} else {
		f, err := os.UserConfigDir()
		if err != nil {
			return "", err
		}
		directory = path.Join(f, "heedy")
	}
	var err error
	directory, err = filepath.Abs(directory)
	if err == nil {
		logrus.Infof("Using database at %s", directory)
	}

	return directory, err
}

func getpid(directory string) (*os.Process, error) {
	b, err := ioutil.ReadFile(path.Join(directory, "heedy.pid"))
	if err != nil {
		return nil, err
	}
	pid, err := strconv.Atoi(string(b))
	if err != nil {
		return nil, err
	}
	return os.FindProcess(pid)
}

func writepid(cdir string) error {
	// First check if the pid exists and is running
	p, err := getpid(cdir)
	if err == nil {
		err = p.Signal(syscall.Signal(0))
		if err == nil && !force_run {
			return fmt.Errorf("Heedy is already running at pid %d", p.Pid)
		}
	}

	// Create pid
	return ioutil.WriteFile(path.Join(cdir, "heedy.pid"), []byte(strconv.Itoa(os.Getpid())), os.ModePerm)
}

func init() {
	RootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Extremely verbose logging of server requests and responses. Only works in DEBUG log level.")
	RootCmd.PersistentFlags().BoolVar(&revert, "revert", false, "Reverts an update from backup if server fails to start")
	RootCmd.PersistentFlags().BoolVar(&force_run, "force", false, "Force the server to start even if it detects a heedy pid running")
}
