package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mock-server/cmd/sftp/internal/consts"
	internal "mock-server/cmd/sftp/internal/hostKey"
	"mock-server/internal/common"

	CharmLog "github.com/charmbracelet/log"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type fileHandler struct{}
type listHandler struct{}
type cmdHandler struct{}

type lister []os.FileInfo

var logger = CharmLog.NewWithOptions(os.Stderr, CharmLog.Options{
	ReportTimestamp: true,
	TimeFormat:      time.Kitchen,
	Prefix:          "SFTP Service 📁",
})

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
func (fs *fileHandler) Fileread(r *sftp.Request) (io.ReaderAt, error) {
	logger.Info("Fileread called", "path", r.Filepath, "method", r.Method)

	cleanPath := filepath.Clean(r.Filepath)
	cleanPath = strings.TrimPrefix(cleanPath, string(filepath.Separator))
	fullPath := filepath.Join(consts.SFTPRoot, cleanPath)

	logger.Info("Reading file", "path", fullPath)
	return os.Open(fullPath)
}

func (fs *fileHandler) Filewrite(r *sftp.Request) (io.WriterAt, error) {
	cleanPath := filepath.Clean(r.Filepath)
	cleanPath = strings.TrimPrefix(cleanPath, string(filepath.Separator))
	fullPath := filepath.Join(consts.SFTPRoot, cleanPath)

	logger.Info("Writing file", "path", fullPath)
	return os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
}

func (fs *cmdHandler) Filecmd(r *sftp.Request) error {
	cleanPath := filepath.Clean(r.Filepath)
	cleanPath = strings.TrimPrefix(cleanPath, string(filepath.Separator))
	fullPath := filepath.Join(consts.SFTPRoot, cleanPath)

	logger.Info("Running command", "method", r.Method, "path", fullPath)
	switch r.Method {
	case "Realpath":
		return nil
	case "Stat", "Lstat", "Fstart":
		stat, err := os.Stat(fullPath)
		if err != nil {
			return err
		}

		logger.Info("Stat result", "isDir", stat.IsDir(), "size", stat.Size())
		return nil
	case "Setstat", "Rename":
		return nil //no-op
	case "Remove":
		return os.Remove(fullPath)
	case "Mkdir":
		return os.Mkdir(fullPath, 0755)
	case "Rmdir":
		return os.RemoveAll(fullPath)
	default:
		// return fmt.Errorf("unsupported method: %s", r.Method)
		return sftp.ErrSshFxOpUnsupported
	}
}

func (fs *listHandler) Filelist(r *sftp.Request) (sftp.ListerAt, error) {
	logger.Info("FileList recieved...", "method", r.Method)
	cleanPath := filepath.Clean(r.Filepath)
	cleanPath = strings.TrimPrefix(cleanPath, string(filepath.Separator))
	fullPath := filepath.Join(consts.SFTPRoot, cleanPath)

	if r.Method == "Stat" {
		logger.Info("Stat request for file", "path", fullPath)
		stat, err := os.Stat(fullPath)
		if err != nil {
			return nil, err
		}

		// Return a single-item lister with just this file's info
		return lister([]os.FileInfo{stat}), nil
	}

	logger.Info("Listing directory", "path", fullPath)
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, err
	}

	var fileInfos []os.FileInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			logger.Warn("Error reading entry", "name", entry.Name(), "error", err)
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

func handleSFTPConnection(netConn net.Conn, config *ssh.ServerConfig) {
	defer netConn.Close()

	sshConn, chnls, reqs, err := ssh.NewServerConn(netConn, config)
	if err != nil {
		logger.Error("SSH handshake failed", err)
		return
	}
	logger.Info("New connection", "sshConn", sshConn.RemoteAddr())
	defer sshConn.Close()

	go ssh.DiscardRequests(reqs)
	for newChannel := range chnls {
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			logger.Error("Could not accept channel", err)
			continue
		}

		go handleSFTPChannel(channel, requests)
	}
}

func handleSFTPChannel(channel ssh.Channel, requets <-chan *ssh.Request) {
	defer channel.Close()

	for req := range requets {
		if req.Type == "subsystem" && string(req.Payload[4:]) == "sftp" {
			if req.WantReply {
				req.Reply(true, nil)
			}

			handlers := sftp.Handlers{
				FileGet:  &fileHandler{},
				FilePut:  &fileHandler{},
				FileCmd:  &cmdHandler{},
				FileList: &listHandler{},
			}

			server := sftp.NewRequestServer(channel, handlers)

			logger.Info("Session started")
			server.Serve()
			logger.Info("Session ended")
			return
		}

		if req.WantReply {
			req.Reply(false, nil)
			logger.Warn("Unsupported request type", "type", req.Type)
			continue
		}
	}
}

func main() {
	hostkey, err := internal.GenerateHostKeyWithLogger(logger)
	if err != nil {
		logger.Fatal("Failed to generate host key", err)
	}

	config := &ssh.ServerConfig{
		PasswordCallback: sftpAuthHandler,
	}
	config.AddHostKey(hostkey)

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", consts.SFTP_PORT))
	if err != nil {
		logger.Fatal("Failed to listen on port", "port", consts.SFTP_PORT, "error", err)
	}
	defer listener.Close()

	logger.Info("SFTP server listening", "port", consts.SFTP_PORT)
	logger.Info("Server Details", "addr", listener.Addr(), "user", "testuser", "password", "testpass")

	for {
		conn, err := listener.Accept()
		if err != nil {
			logger.Warn("Failed to accept connection", err)
			continue
		}

		go handleSFTPConnection(conn, config)
		logger.Info("Accepted new connection", "remoteAddr", conn.RemoteAddr())
	}
}
