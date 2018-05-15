package ssh

import (
	boshdir "github.com/cloudfoundry/bosh-cli/director"
	boshssh "github.com/cloudfoundry/bosh-cli/ssh"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
	"os"
)

type CustomNonInteractiveRunner struct {
	customRunner CustomRunner
}

func NewCustomNonInteractiveRunner(customRunner CustomRunner) CustomNonInteractiveRunner {
	return CustomNonInteractiveRunner{customRunner}
}

func (r CustomNonInteractiveRunner) Run(connOpts boshssh.ConnectionOpts, result boshdir.SSHResult, rawCmd []string) error {
	if len(result.Hosts) == 0 {
		return bosherr.Errorf("Non-interactive SSH expects at least one host")
	}
	if len(rawCmd) == 0 {
		return bosherr.Errorf("Non-interactive SSH expects non-empty command")
	}
	cmdFactory := func(host boshdir.Host, sshArgs boshssh.SSHArgs) boshsys.Command {
		return boshsys.Command{
			Name:         "ssh",
			Args:         append(append(sshArgs.OptsForHost(host), sshArgs.LoginForHost(host)...), rawCmd...),
			KeepAttached: true,
			Stdin:        os.Stdin,
		}
	}

	return r.customRunner.Run(connOpts, result, cmdFactory)
}
