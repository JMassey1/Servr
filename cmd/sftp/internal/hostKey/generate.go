package generate

import (
	"os"
	"path/filepath"

	"mock-server/cmd/sftp/internal/consts"

	CharmLog "github.com/charmbracelet/log"
	"golang.org/x/crypto/ssh"
)

func GenerateHostKey() (ssh.Signer, error) {
	keyBytes, err := os.ReadFile(filepath.Join(consts.SFTPRoot, "ssh", "id_rsa_mockapi"))
	if err != nil {
		return nil, err
	}

	return ssh.ParsePrivateKey(keyBytes)
}

func GenerateHostKeyWithLogger(loggerParent *CharmLog.Logger) (ssh.Signer, error) {
	logger := loggerParent.WithPrefix("HostKey Generation")
	keyBytes, err := os.ReadFile(filepath.Join(consts.SFTPRoot, "ssh", "id_rsa_mockapi"))
	if err != nil {
		logger.Error("Failed to read host key file", err)
		return nil, err
	}

	return ssh.ParsePrivateKey(keyBytes)
}
