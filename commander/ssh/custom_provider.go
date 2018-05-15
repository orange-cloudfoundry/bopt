package ssh

import (
	boshdir "github.com/cloudfoundry/bosh-cli/director"
	boshssh "github.com/cloudfoundry/bosh-cli/ssh"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
	"io"
)

type CustomProvider struct {
	customRunner CustomRunner
}

func NewCustomProvider(cmdRunner boshsys.CmdRunner, fs boshsys.FileSystem, writerStdout io.Writer, writerStderr io.Writer, logger boshlog.Logger) CustomProvider {
	sshSessionFactory := func(connOpts boshssh.ConnectionOpts, result boshdir.SSHResult) boshssh.Session {
		return boshssh.NewSessionImpl(connOpts, boshssh.SessionImplOpts{ForceTTY: true}, result, fs)
	}

	customSsh := NewCustomRunner(cmdRunner, sshSessionFactory, writerStdout, writerStderr, logger)

	return CustomProvider{customRunner: customSsh}
}

func (p CustomProvider) NewSSHRunner() boshssh.Runner {
	return NewCustomNonInteractiveRunner(p.customRunner)
}
