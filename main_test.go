package main

import (
	"os"
	"path/filepath"
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

	arguments := Arguments {
		cwd: "",
		paths: paths,
		command: "",
	}
	
	go CheckForChange(func(_ string) { checked <- true }, arguments)

	time.Sleep(time.Millisecond * 10)

	currentTime := time.Now().Local()
	err = os.Chtimes(file.Name(), currentTime, currentTime)
	check(t, err, "Failed to change times!")

	select {
	case <-checked:
		return
	case <-time.After(20 * time.Millisecond):
		t.Errorf("Failed to see file change")
		return
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
	// paths are already absolute
	go CheckForChange(func(_ string) { checked <- true }, arguments)

	time.Sleep(time.Millisecond * 10)

	currentTime := time.Now().Local()
	err = os.Chtimes(file3.Name(), currentTime, currentTime)
	check(t, err, "Failed to change times!")

	select {
	case <-checked:
		return
	case <-time.After(20 * time.Millisecond):
		t.Errorf("Failed to see file change")
		return
	}
}

func TestAddChildren(t *testing.T) {
	dir, err := os.Stat("test_folder")
	check(t, err, "Failed to stat dir!")

	cwd, err := os.Getwd()
	check(t, err, "Failed to get cwd!")

	children, err := addChildren(filepath.Join(cwd, dir.Name()), dir, []string{})
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

	children, err := addChildren(filepath.Join(cwd, dir.Name()), dir, []string{".txt", ".csv"})
	check(t, err, "Failed to get children!")

	if len(children) != 2 {
		t.Errorf("Got children wrong! %s", children)
	}
}