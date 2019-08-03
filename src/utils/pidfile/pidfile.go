package pidfile

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"syscall"
)

func WritePidFile(path string) error {
	_, err := os.Stat(path)

	if err == nil { // file already exists
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return fmt.Errorf("Could not read %s: %v", path, err)
		}

		pid, err := strconv.Atoi(string(data))
		if err != nil {
			return fmt.Errorf("Could not parse pid file %s contents '%s': %v", path, string(data), err)
		}

		if process, err := os.FindProcess(pid); err == nil {
			if err := process.Signal(syscall.Signal(0)); err == nil {
				return fmt.Errorf("process with pid %d is still running", pid)
			}
		}
	}

	return ioutil.WriteFile(path, []byte(fmt.Sprintf("%d", os.Getpid())), 0664)

}
