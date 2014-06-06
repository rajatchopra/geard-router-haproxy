package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"strings"
	//"bufio"
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

func commandServer(c net.Conn) {
	for {
		buf := make([]byte, 512)
		nr, err := c.Read(buf)
		if err != nil {
			return
		}

		data := strings.Trim(string(buf[0:nr]), "\n ")
		println("Server got:", data)
		var return_str (string)
		if data == "Start" {
			return_str = ":OK\n"
			cmd := exec.Command("/var/lib/haproxy/bin/write_haproxy_config")
			execCmd(cmd)
			cmd = exec.Command("haproxy", "-f", "/var/lib/haproxy/conf/haproxy.config", "-p", "/var/lib/haproxy/run/haproxy.pid")
			execCmd(cmd)
		} else if data == "Stop" {
			return_str = ":OK\n"
			cmd := exec.Command("pkill", "haproxy")
			execCmd(cmd)
		} else if data == "Reload" {
			return_str = ":OK\n"
			cmd := exec.Command("/var/lib/haproxy/bin/write_haproxy_config")
			out,status := execCmd(cmd)
			if !status {
				_, err = c.Write([]byte(out))
				continue
			}
			old_pid, oerr := ioutil.ReadFile("/var/lib/haproxy/run/haproxy.pid")
			if oerr != nil {
				cmd = exec.Command("haproxy", "-f", "/var/lib/haproxy/conf/haproxy.config", "-p", "/var/lib/haproxy/run/haproxy.pid")
			} else {
				cmd = exec.Command("haproxy", "-f", "/var/lib/haproxy/conf/haproxy.config", "-p", "/var/lib/haproxy/run/haproxy.pid", "-sf", string(old_pid))
			}
			out,status = execCmd(cmd)
			return_str = return_str + out
		} else {
			return_str = "Command should be one of 'Start/Stop/Reload'"
		}
		fmt.Printf("Writing %s to socket\n", return_str)
		_, err = c.Write([]byte(return_str))
		if err != nil {
			println("Write: ", err.Error())
			//panic("Write Error.")
		}
	}
}

func main() {
	filename := "/var/lib/haproxy/run/router_interface.sock"
	rerr := os.Remove(filename)
	if rerr != nil {
		println("Failed to remove unix socket file", rerr)
		os.Exit(1)
	}
	l, err := net.Listen("unix", filename)
	if err != nil {
		println("listen error", err)
		os.Exit(1)
	}

	for {
		fd, err := l.Accept()
		if err != nil {
			println("accept error", err)
			os.Exit(1)
		}

		go commandServer(fd)
	}
}
