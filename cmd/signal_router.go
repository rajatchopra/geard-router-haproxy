package main

import (
	"fmt"
	"os"
	"os/signal"
	"os/exec"
	"syscall"
	"io/ioutil"
)


func execCmd(cmd *exec.Cmd) (string, bool) {
	out, err := cmd.CombinedOutput()
	var return_str string
	if err != nil {
		fmt.Sprintf(return_str, "Error executing command.\n%s", err.Error())
	} else {
		return_str = string(out)
	}
	return return_str, err==nil
}

func ReloadHaproxy() {
	cmd := exec.Command("/usr/bin/geard-router-haproxy-writeconfig")
	out,status := execCmd(cmd)
	if !status {
		fmt.Println("Failed to correctly create haproxy config.")
	}
	old_pid, oerr := ioutil.ReadFile("/var/lib/haproxy/run/haproxy.pid")
	if oerr != nil {
		cmd = exec.Command("haproxy", "-f", "/var/lib/haproxy/conf/haproxy.config", "-p", "/var/lib/haproxy/run/haproxy.pid")
	} else {
		cmd = exec.Command("haproxy", "-f", "/var/lib/haproxy/conf/haproxy.config", "-p", "/var/lib/haproxy/run/haproxy.pid", "-sf", string(old_pid))
	}
	out,status = execCmd(cmd)
	fmt.Println(out)
}

func main() {
	// Set up channel on which to send signal notifications.
	// We must use a buffered channel or risk missing the signal
	// if we're not ready to receive when the signal is sent.
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGUSR2, syscall.SIGUSR1)

	// Block until a signal is received.
	for {
		s := <-c
		fmt.Println("Got signal:", s)
		ReloadHaproxy()
	}
}

