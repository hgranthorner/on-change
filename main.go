package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	args := os.Args[1:]
	fmt.Println(args)

	if len(args) == 0 ||
		args[0] == "-h" ||
		args[0] == "--help" {
		printHelp()
	}

	channel := make(chan bool)

	go checkForChange(func() { channel <- true }, []string{"go.mod", "main.go"})

	for <-channel {
		fmt.Println("change")
	}
}

// Runs continuously, calling callbackFn whenever any of the files in
// passed in array changes.
func checkForChange(callbackFn func(), paths []string) {
	for _, path := range paths {
		res, err := os.Stat(path)
		if err != nil {
			fmt.Println(err)
			return
		}
		timeChanged := res.ModTime()

		go checkForFileChange(callbackFn, path, timeChanged)
	}
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
