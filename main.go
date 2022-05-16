package main

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

type Arguments struct {
	extensions []string
	cwd        string
	command    string
	paths      []string
	exclusions []*regexp.Regexp
	verbose bool
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
			extensions := strings.Split(args[i+1], ",")
			arguments.extensions = append(arguments.extensions, extensions...)
			argIndicesToSkip = append(argIndicesToSkip, i, i+1)
		}
		if (arg == "--exclude" || arg == "-exc") &&
			i < len(args) {
			exclusions := strings.Split(args[i+1], ",")
			for _, exc := range exclusions {
				regex, err := regexp.Compile(exc)
				if err != nil {
					fmt.Println("Failed to compile", exc, "as regular expression")
					return
				}
				arguments.exclusions = append(arguments.exclusions, regex)
			}
			argIndicesToSkip = append(argIndicesToSkip, i, i+1)
		}
		if arg == "-v" || arg == "--verbose" {
			arguments.verbose = true
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
	quit := make(chan bool)

	go CheckForChange(func(path string) { channel <- path }, quit, arguments)

	for {
		select {
		case changedFile := <-channel:
			parts := strings.Split(changedFile, string(filepath.Separator))
			fileName := parts[len(parts) - 1]
			fmt.Println("Change detected: ", fileName)
			RunCommand(arguments.command)
		case <-quit:
			return
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
func CheckForChange(callbackFn func(string), quit chan<- bool, arguments Arguments) {
	filePaths := []string{}

	// first build list of all files to watch by searching through directories
	for _, path := range arguments.paths {
		file, err := os.Stat(path)
		if err != nil {
			panic(err)
		}

		if file.IsDir() {
			children, err := addChildren(
				filepath.Join(arguments.cwd, file.Name()),
				file,
				arguments.extensions,
				arguments.exclusions,
			)
			if err != nil {
				panic(err)
			}

			filePaths = append(filePaths, children[:]...)
			continue
		}

		filePaths = maybeAppend(
			filePaths,
			filepath.Join(arguments.cwd, path),
			arguments.extensions,
			arguments.exclusions,
		)
	}

	if arguments.verbose {
		fmt.Println("Watching: ", filePaths)
	}

	if len(filePaths) == 0 {
		fmt.Println("Passed parameters match no files!")
		quit <- true

	}
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
func addChildren(parentPath string, parent fs.FileInfo, extensions []string, exclusions []*regexp.Regexp) ([]string, error) {
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
			newFiles = maybeAppend(newFiles, path.Value, extensions, exclusions)
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

func maybeAppend(arr []string, s string, extensions []string, exclusions []*regexp.Regexp) []string {
	hasExtension := len(extensions) == 0
	anyExclusions := false

	for _, extension := range extensions {
		if strings.HasSuffix(s, extension) {
			hasExtension = true
			break
		}
	}

	for _, exclusion := range exclusions {
		if exclusion.MatchString(s) {
			anyExclusions = true
			break
		}
	}

	if !anyExclusions && hasExtension {
		arr = append(arr, s)
	}

	return arr
}

func printHelp() {
	fmt.Println("a command line utility for rerunning a command on file change.")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("")
	fmt.Println("on-change <cmd> <dir> (-ext|--extension <comma separated extensions>) (-exc|--exclude <comma separated file names to exclude>")
	fmt.Println("")
	fmt.Println("where <cmd> is some command line program ('ls'), <dir> is the directory to watch ('.', './src') and globs are a comma separated list of regexes that represent file paths.")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("")
	fmt.Println("run ls whenever a file in the current directory changes")
	fmt.Println("`on-change ls .`")
	fmt.Println("")
	fmt.Println("run ls whenever a javascript file changes")
	fmt.Println("`on-change ls src/ -ext .js`")
	fmt.Println("")
	fmt.Println("run ls whenever a javascript or typescript file changes")
	fmt.Println("`on-change ls src/ -ext .js,.ts`")
	fmt.Println("run ls whenever a javascript or typescript file changes, except for foo.js")
	fmt.Println("`on-change ls src/ -ext .js,.ts -exc foo.js`")
}
