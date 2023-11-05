package main

import (
	"flag"
	"fmt"
	"io/fs"
	"net/mail"
	"os"
	"path/filepath"
	"time"
)

var fix bool = false

type MailFileInfo struct {
	Path     string
	FSTime   time.Time
	Mailtime time.Time
}

func (mfi *MailFileInfo) AbsDiff() time.Duration {
	return mfi.Mailtime.Sub(mfi.FSTime).Abs()
}

func (mfi *MailFileInfo) SetFSTime() error {
	return os.Chtimes(mfi.Path, time.Time{}, mfi.Mailtime)
}

func (mfi *MailFileInfo) Fix() error {
	diff := mfi.AbsDiff().Hours()
	if diff > 1.0 {
		fmt.Printf("%s -> %s\t%s\n", mfi.FSTime.UTC(), mfi.Mailtime.UTC(), mfi.Path)
		if fix {
			return mfi.SetFSTime()
		}
	}
	return nil
}

func NewMailFileInfoFromFileInfo(path string, fileinfo fs.FileInfo) (*MailFileInfo, error) {
	fstime := fileinfo.ModTime()

	r, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	msg, err := mail.ReadMessage(r)
	if err != nil {
		return nil, err
	}
	mailtime, err := msg.Header.Date()
	if err != nil {
		return nil, err
	}

	return &MailFileInfo{
		Path:     path,
		FSTime:   fstime,
		Mailtime: mailtime,
	}, nil
}

func WalkMaildirCur(root string, fn fs.WalkDirFunc) error {
	return filepath.WalkDir(root, func(path string, direntry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !direntry.Type().IsRegular() {
			return nil
		}
		if filepath.Base(filepath.Dir(path)) != "cur" {
			return nil
		}
		return fn(path, direntry, nil)
	})
}

func FixMaildir(root string) error {
	return WalkMaildirCur(root, func(path string, direntry fs.DirEntry, walkErr error) error {
		var err error
		defer func() {
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v [%s]\n", err, path)
			}
		}()
		fileInfo, err := direntry.Info()
		if err != nil {
			return nil
		}
		mfi, err := NewMailFileInfoFromFileInfo(path, fileInfo)
		if err != nil {
			return nil
		}
		err = mfi.Fix()
		return nil
	})
}

func main() {
	flag.BoolVar(&fix, "fix", false, "fix mtime")
	flag.Parse()
	for _, root := range flag.Args() {
		err := FixMaildir(root)
		if err != nil {
			fmt.Fprintf(os.Stderr, "fatal error: %s\n", err)
			os.Exit(1)
		}
	}
}
