package main

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"
)

func check(t *testing.T, err error, msg string) {
	if err != nil {
		t.Errorf(msg)
	}
}

func TestCheckForChangeInOneFile(t *testing.T) {
	file, err := os.CreateTemp("", "newfile")
	check(t, err, "Failed to create temporary file!")
	defer file.Close()

	paths := []string{file.Name()}
	checked := make(chan bool)
	quit := make(chan bool)

	arguments := Arguments {
		cwd: "",
		paths: paths,
		command: "",
	}
	
	go CheckForChange(func(_ string) { checked <- true }, quit, arguments)

	time.Sleep(time.Millisecond * 10)

	currentTime := time.Now().Local()
	err = os.Chtimes(file.Name(), currentTime, currentTime)
	check(t, err, "Failed to change times!")

	select {
	case <-checked:
		return
	case <-time.After(20 * time.Millisecond):
		t.Errorf("Failed to see file change")
	case <-quit:
		t.Errorf("Failed to read any files")
	}
}

func TestCheckForChangeInSeveralFiles(t *testing.T) {
	file1, err := os.CreateTemp("", "newfile")
	file2, err := os.CreateTemp("", "newfile")
	file3, err := os.CreateTemp("", "newfile")

	check(t, err, "Failed to create temporary file!")

	defer file1.Close()
	defer file2.Close()
	defer file3.Close()

	arguments := Arguments {
		cwd: "",
		paths: []string{file1.Name(), file2.Name(), file3.Name()},
		command: "",
	}
	checked := make(chan bool)
	quit := make(chan bool)
	// paths are already absolute
	go CheckForChange(func(_ string) { checked <- true }, quit, arguments)

	time.Sleep(time.Millisecond * 10)

	currentTime := time.Now().Local()
	err = os.Chtimes(file3.Name(), currentTime, currentTime)
	check(t, err, "Failed to change times!")

	select {
	case <-checked:
		return
	case <-time.After(20 * time.Millisecond):
		t.Errorf("Failed to see file change")
	case <-quit:
		t.Errorf("Failed to read any files")
	}
}

func TestAddChildren(t *testing.T) {
	dir, err := os.Stat("test_folder")
	check(t, err, "Failed to stat dir!")

	cwd, err := os.Getwd()
	check(t, err, "Failed to get cwd!")

	children, err := addChildren(filepath.Join(cwd, dir.Name()), dir, []string{}, []*regexp.Regexp{})
	check(t, err, "Failed to get children!")

	if len(children) != 3 {
		t.Errorf("Got children wrong! %s", children)
	}
}

func TestAddChildrenWithExtensionFilter(t *testing.T) {
	dir, err := os.Stat("test_folder")
	check(t, err, "Failed to stat dir!")

	cwd, err := os.Getwd()
	check(t, err, "Failed to get cwd!")

	children, err := addChildren(filepath.Join(cwd, dir.Name()), dir, []string{".txt", ".csv"}, []*regexp.Regexp{})
	check(t, err, "Failed to get children!")

	if len(children) != 2 {
		t.Errorf("Got children wrong! %s", children)
	}
}

func TestMaybeAppendWithNoExtensionsOrExceptions(t *testing.T) {
	arr := []string{"egg/salad.txt"}
	// regex, err := regexp.Compile("csv")
	// check(t, err, "Failed to compile regex")
	// exclusions := []*regexp.Regexp{regex}
	arr = maybeAppend(arr, "egg/sandwich.csv", []string{}, []*regexp.Regexp{})
	if len(arr) != 2 {
		t.Errorf("Failed to append to array!")
	}
}

func TestMaybeAppendWithExtensionsAndNoExceptions(t *testing.T) {
	arr := []string{"egg/salad.txt"}

	arr = maybeAppend(arr, "egg/sandwich.csv", []string{".xlsx"}, []*regexp.Regexp{})
	if len(arr) != 1 {
		t.Errorf("Appended to array when it shouldn't have.")
	}

	arr = maybeAppend(arr, "egg/sandwich.csv", []string{".csv"}, []*regexp.Regexp{})
	if len(arr) != 2 {
		t.Errorf("Failed to append to array!")
	}
}

func TestMaybeAppendWithNoExtensionsAndSomeExceptions(t *testing.T) {
	arr := []string{"egg/salad.txt"}

	regex, err := regexp.Compile("egg")
	check(t, err, "Failed to compile regex")
	exclusions := []*regexp.Regexp{regex}

	arr = maybeAppend(arr, "egg/sandwich.csv", []string{}, exclusions)
	if len(arr) != 1 {
		t.Errorf("Appended to array when it shouldn't have.")
	}

	arr = maybeAppend(arr, "ham/sandwich.csv", []string{}, exclusions)
	if len(arr) != 2 {
		t.Errorf("Didn't append to array when it should have.")
	}
}