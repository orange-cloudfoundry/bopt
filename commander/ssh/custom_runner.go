package ssh

import (
	"fmt"
	boshdir "github.com/cloudfoundry/bosh-cli/director"
	boshssh "github.com/cloudfoundry/bosh-cli/ssh"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
	"io"
)

type CustomRunner struct {
	cmdRunner      boshsys.CmdRunner
	sessionFactory func(boshssh.ConnectionOpts, boshdir.SSHResult) boshssh.Session

	writerStdout io.Writer
	writerStderr io.Writer

	logTag string
	logger boshlog.Logger
}

func NewCustomRunner(
	cmdRunner boshsys.CmdRunner,
	sessionFactory func(boshssh.ConnectionOpts, boshdir.SSHResult) boshssh.Session,
	writerStdout io.Writer,
	writerStderr io.Writer,
	logger boshlog.Logger,
) CustomRunner {
	return CustomRunner{
		cmdRunner:      cmdRunner,
		sessionFactory: sessionFactory,

		writerStdout: writerStdout,
		writerStderr: writerStderr,

		logTag: "CustomRunner",
		logger: logger,
	}
}

func (r CustomRunner) Run(connOpts boshssh.ConnectionOpts, result boshdir.SSHResult, cmdFactory func(boshdir.Host, boshssh.SSHArgs) boshsys.Command) error {
	sess := r.sessionFactory(connOpts, result)

	sshOpts, err := sess.Start()
	if err != nil {
		return bosherr.WrapErrorf(err, "Setting up SSH session")
	}

	defer func() {
		_ = sess.Finish()
	}()

	cmds := r.makeCmds(result.Hosts, sshOpts, cmdFactory)

	return r.runCmds(cmds)
}

type customRunnerCmd struct {
	boshsys.Command
}

func (r CustomRunner) makeCmds(hosts []boshdir.Host, sshOpts boshssh.SSHArgs, cmdFactory func(boshdir.Host, boshssh.SSHArgs) boshsys.Command) []customRunnerCmd {
	var cmds []customRunnerCmd

	for _, host := range hosts {
		cmd := cmdFactory(host, sshOpts)
		cmds = append(cmds, customRunnerCmd{cmd})
	}

	return cmds
}

func (r CustomRunner) runCmds(cmds []customRunnerCmd) error {
	for _, cmd := range cmds {
		stdout, stderr, exitCode, err := r.cmdRunner.RunComplexCommand(cmd.Command)
		if err != nil {
			return NewErrCommandSsh(err.Error(), exitCode)
		}
		fmt.Fprint(r.writerStdout, stdout)
		fmt.Fprint(r.writerStderr, stderr)
	}
	return nil
}
