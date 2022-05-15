package main

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

func main() {
	args := os.Args[1:]
	fmt.Println(args)

	if len(args) == 0 ||
		args[0] == "-h" ||
		args[0] == "--help" {
		printHelp()
		return
	}

	channel := make(chan bool)

	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	// go CheckForChange(func() { channel <- true }, []string{"go.mod", "main.go"})
	go CheckForChange(cwd, func() { channel <- true }, []string{"./test_folder"})

	for {
		select {
		case <-channel:
			fmt.Println("change")
		}
	}
}

// Runs continuously, calling callbackFn whenever any of the files in
// passed in array changes.
func CheckForChange(cwd string, callbackFn func(), paths []string) {
	filePaths := []string{}

	// first build list of all files to watch by searching through directories
	for _, path := range paths {
		file, err := os.Stat(path)
		if err != nil {
			panic(err)
		}

		if file.IsDir() {
			children, err := addChildren(filepath.Join(cwd, file.Name()), file)
			if err != nil {
				panic(err)
			}

			filePaths = append(filePaths, children[:]...)
		} else {
			filePaths = append(filePaths, filepath.Join(cwd, path))
		}
	}

	fmt.Println(filePaths)
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
func addChildren(parentPath string, parent fs.FileInfo) ([]string, error) {
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
			newFiles = append(newFiles, path.Value)
		}

		n++
	}

	return newFiles, nil
}

func checkForFileChange(callbackFn func(), path string, timeChanged time.Time) {
	for {
		time.Sleep(20 * time.Millisecond)

		res, err := os.Stat(path)
		if err != nil {
			fmt.Println(err)
			return
		}
		newTimeChanged := res.ModTime()
		if newTimeChanged.After(timeChanged) {
			callbackFn()
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
