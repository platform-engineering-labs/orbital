package sys

import (
	"os"
	"os/exec"
	"syscall"
)

func IsPrivilegedUser() bool {
	return os.Geteuid() == 0
}

func InvokeSelfWithSudo(args ...string) error {
	if len(args) == 0 {
		args = os.Args[1:]
	}

	self, err := os.Executable()
	if err != nil {
		return err
	}

	sudo, err := exec.LookPath("sudo")
	if err != nil {
		return err
	}

	env := os.Environ()
	path := os.Getenv("PATH")

	args = append([]string{" ", "env", "PATH=" + path, self}, args...)

	err = syscall.Exec(sudo, args, env)
	if err != nil {
		return err
	}

	return nil
}

func SudoSessionActive() bool {
	sudo, err := exec.LookPath("sudo")
	if err != nil {
		return false
	}

	cmd := exec.Command(sudo, "-n", "true")

	err = cmd.Run()
	if err != nil {
		return false
	}

	return true
}
