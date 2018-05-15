package ssh

import "fmt"

type ErrCommandSsh struct {
	message  string
	exitCode int
}

func NewErrCommandSsh(message string, exitCode int) error {
	return ErrCommandSsh{
		message:  message,
		exitCode: exitCode,
	}
}
func (e ErrCommandSsh) ExitCode() int {
	return e.exitCode
}
func (e ErrCommandSsh) Error() string {
	return e.message
}
func (e ErrCommandSsh) String() string {
	return fmt.Sprintf("Command exit with code %d: %s", e.exitCode, e.message)
}
