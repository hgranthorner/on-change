package main

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type Arguments struct {
	extension string
	cwd       string
	command   string
	paths     []string
}

func main() {
	args := os.Args[1:]
	fmt.Println(args)

	if len(args) == 0 ||
		args[0] == "-h" ||
		args[0] == "--help" ||
		args[0] == "--extension" ||
		args[0] == "-ext" {
		printHelp()
		return
	}

	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	arguments := Arguments{
		command: args[0],
		cwd:     cwd,
	}
	argIndicesToSkip := []int{}

	for i, arg := range args {
		if (arg == "--extension" || arg == "-ext") &&
			i < len(args) {
			arguments.extension = args[i+1]
			argIndicesToSkip = append(argIndicesToSkip, i, i+1)
		}
	}

	for i, f := range args {
		// skip first argument (the command) and any optional arguments
		if i == 0 ||
			contains(argIndicesToSkip, i) != -1 {
			continue
		}
		arguments.paths = append(arguments.paths, f)
	}

	channel := make(chan string)

	go CheckForChange(func(path string) { channel <- path }, arguments)

	for {
		select {
		case changedFile := <-channel:
			fmt.Println("Change detected: ", changedFile)
			RunCommand(arguments.command)
		}
	}
}

func contains[T comparable](arr []T, x T) int {
	for i, val := range arr {
		if val == x {
			return i
		}
	}

	return -1
}

func RunCommand(command string) {
	os := runtime.GOOS
	var cmd *exec.Cmd
	if os == "windows" {
		cmd = exec.Command(command)
	} else {
		cmdWithArgs := strings.Split(command, " ")
		cmd = exec.Command(cmdWithArgs[0], cmdWithArgs[1:]...)
	}
	stdout, err := cmd.Output()

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	// Print the output
	fmt.Println(string(stdout))
}

// Runs continuously, calling callbackFn whenever any of the files in
// passed in array changes.
func CheckForChange(callbackFn func(string), arguments Arguments) {
	filePaths := []string{}

	// first build list of all files to watch by searching through directories
	for _, path := range arguments.paths {
		file, err := os.Stat(path)
		if err != nil {
			panic(err)
		}

		if file.IsDir() {
			children, err := addChildren(filepath.Join(arguments.cwd, file.Name()), file, arguments.extension)
			if err != nil {
				panic(err)
			}

			filePaths = append(filePaths, children[:]...)
		} else {
			filePaths = append(filePaths, filepath.Join(arguments.cwd, path))
		}
	}

	fmt.Println("Watching: ", filePaths)
	// then actually set up the watch
	for _, path := range filePaths {
		res, err := os.Stat(path)
		if err != nil {
			panic(err)
		}
		timeChanged := res.ModTime()

		go checkForFileChange(callbackFn, path, timeChanged)
	}
}

type AbsolutePath struct {
	Value string
	IsDir bool
}

func AbsolutePathFromFileInfo(parentPath string, info fs.FileInfo) AbsolutePath {
	return AbsolutePath{
		filepath.Join(parentPath, info.Name()),
		info.IsDir(),
	}
}

// Returns a list of absolute paths to files
func addChildren(parentPath string, parent fs.FileInfo, extension string) ([]string, error) {
	newFiles := []string{}
	tempFiles, err := ioutil.ReadDir(parentPath)
	filePaths := []AbsolutePath{}
	if err != nil {
		fmt.Println("could not open directory", parentPath)
		return nil, err
	}

	// convert fileinfos to absolute paths
	for _, f := range tempFiles {
		filePaths = append(filePaths, AbsolutePathFromFileInfo(parentPath, f))
	}

	n := 0
	for n < len(filePaths) {
		path := filePaths[n]

		if path.IsDir {
			dir, err := ioutil.ReadDir(path.Value)
			if err != nil {
				fmt.Println("could not open directory", path.Value)
				return nil, err
			}

			for _, f := range dir {
				filePaths = append(filePaths, AbsolutePathFromFileInfo(path.Value, f))
			}
		} else {
			if extension == "" || strings.HasSuffix(path.Value, extension) {
				newFiles = append(newFiles, path.Value)
			}
		}

		n++
	}

	return newFiles, nil
}

func checkForFileChange(callbackFn func(string), path string, timeChanged time.Time) {
	for {
		time.Sleep(20 * time.Millisecond)

		res, err := os.Stat(path)
		if err != nil {
			fmt.Println(err)
			return
		}
		newTimeChanged := res.ModTime()
		if newTimeChanged.After(timeChanged) {
			callbackFn(path)
		}

		timeChanged = newTimeChanged
	}
}

func printHelp() {
	fmt.Println("a command line utility for rerunning a command on file change.")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("")
	fmt.Println("on-change <cmd> <dir> (-i|--include globs)(-e|--exclude globs")
	fmt.Println("")
	fmt.Println("where <cmd> is some command line program ('ls'), <dir> is the directory to watch ('.', './src') and globs are a comma separated list of regexes that represent file paths.")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("")
	fmt.Println("run ls whenever a file in the current directory changes")
	fmt.Println("`on-change ls .`")
}
