package conply

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"syscall"
	"time"
)

// Convert seconds to "mm:ss" time format.
func FormatTime(s uint64) string {
	min := s / 60
	sec := s % 60
	format := "%d:%d"
	if sec < 10 {
		format = "%d:0%d"
	}
	return fmt.Sprintf(format, min, sec)
}

// Checks is file or directory exists.
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func FileAge(path string) time.Duration {
	now := time.Now()
	fi, err := os.Stat(path)
	if err != nil {
		return 7*24*3600 + 1
	}
	modTime := fi.ModTime()
	return time.Duration(now.Sub(modTime).Seconds())
}

// Tries to create the directory.
func Mkdir(path string) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return err
	}
	return nil
}

// Write data to file. If file doesn't exists, it will create it before.
func FilePut(path, data string) error {
	if _, err := os.Create(path); err != nil {
		return err
	}

	file, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()
	if _, err = file.WriteString(data); err != nil {
		return err
	}
	if err = file.Sync(); err != nil {
		return err
	}

	return nil
}

// Read file contents.
func FilePull(path string) (string, error) {
	raw, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// Download the file and report about any error.
func FileDl(url, dest string) (err error) {
	var (
		fh *os.File
		resp *http.Response
	)
	fh, err = os.Create(dest)
	if err != nil {
		return
	}
	defer func() {
		err = fh.Close()
		if err != nil {
			return
		}
	}()

	resp, err = http.Get(url)
	if err != nil {
		return
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			return
		}
	}()

	_, err = io.Copy(fh, resp.Body)
	if err != nil {
		return
	}

	return
}

// Send SIGTERM signal and finish working.
func Halt(code int) error {
	err := syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
	os.Exit(code)
	return err
}
