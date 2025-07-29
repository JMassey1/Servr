package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"mock-server/internal/common"
	"mock-server/internal/consts"

	"github.com/charmbracelet/log"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type realFS struct{}
type lister []os.FileInfo

func sftpAuthHandler(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
	user, err := common.ValidateAuth(string(password))
	if err != nil {
		return nil, err
	}

	return &ssh.Permissions{
		Extensions: map[string]string{
			"user":  user.Username,
			"email": user.Email,
		},
	}, nil
}

// Mock File System Implementation
func (fs *realFS) Fileread(r *sftp.Request) (io.ReaderAt, error) {
	fullPath := filepath.Join(consts.SFTP_ROOT, r.Filepath)
	log.Info("SFTP Read: %s", fullPath)
	return os.Open(fullPath)
}

func (fs *realFS) Filewrite(r *sftp.Request) (io.WriterAt, error) {
	fullPath := filepath.Join(consts.SFTP_ROOT, r.Filepath)
	log.Info("SFTP Write: %s", fullPath)
	return os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
}

func (fs *realFS) Filecmd(r *sftp.Request) error {
	fullPath := filepath.Join(consts.SFTP_ROOT, r.Filepath)
	log.Info("SFTP Command: %s %s", r.Method, fullPath)

	switch r.Method {
	case "Setstat", "Rename":
		return nil //no-op
	case "Remove":
		return os.Remove(fullPath)
	case "Mkdir":
		return os.Mkdir(fullPath, 0755)
	case "Rmdir":
		return os.RemoveAll(fullPath)
	default:
		return fmt.Errorf("unsupported method: %s", r.Method)
	}
}

func (fs *realFS) Filelist(r *sftp.Request) (sftp.ListerAt, error) {
	fullPath := filepath.Join(consts.SFTP_ROOT, r.Filepath)
	log.Info("SFTP List: %s", fullPath)

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, err
	}

	var fileInfos []os.FileInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			log.Printf("SFTP List: Error reading entry %s: %v", entry.Name(), err)
			continue
		}
		fileInfos = append(fileInfos, info)
	}

	return lister(fileInfos), nil
}

// Mock Lister Implementation
func (l lister) ListAt(f []os.FileInfo, off int64) (int, error) {
	if off >= int64(len(l)) {
		return 0, io.EOF
	}

	n := copy(f, l[off:])
	if int(off)+n >= len(l) {
		return n, io.EOF
	}

	return n, nil
}
