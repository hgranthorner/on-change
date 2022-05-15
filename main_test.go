package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCheckForChangeInOneFile(t *testing.T) {
	file, err := os.CreateTemp("", "newfile")

	if err != nil {
		t.Errorf("Failed to create temporary directory!")
		return
	}
	defer file.Close()

	paths := []string{file.Name()}
	checked := make(chan bool)
	
	go CheckForChange("", func() { checked <- true }, paths)

	time.Sleep(time.Millisecond * 10)

	currentTime := time.Now().Local()
	err = os.Chtimes(file.Name(), currentTime, currentTime)
	if err != nil {
		t.Errorf("Failed to change times!")
		return
	}

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

	if err != nil {
		t.Errorf("Failed to create temporary directory!")
		return
	}
	defer file1.Close()
	defer file2.Close()
	defer file3.Close()

	paths := []string{file1.Name(), file2.Name(), file3.Name()}
	checked := make(chan bool)
	// paths are already absolute
	go CheckForChange("", func() { checked <- true }, paths)

	time.Sleep(time.Millisecond * 10)

	currentTime := time.Now().Local()
	err = os.Chtimes(file3.Name(), currentTime, currentTime)
	if err != nil {
		t.Errorf("Failed to change times!")
		return
	}

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
	if err != nil {
		t.Errorf("Failed to stat dir!")
		return
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Errorf("Failed to get cwd!")
		return
	}
	children, err := addChildren(filepath.Join(cwd, dir.Name()), dir)
	if err != nil {
		t.Errorf("Failed to get children!")
		return
	}

	if len(children) != 3 {
		t.Errorf("Got children wrong! %s", children)
		return
	}
}
