package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type Gear struct {
	Name     string
	Location string
}

type Application struct {
	Name  string
	Fqdn  string
	Image string
	Gears []Gear
}

var GlobalAppMap map[string]Application

const (
	AppData = "/var/lib/containers/router/apps.json"
)

func minister(c net.Conn) {
	for {
		buf := make([]byte, 1024)
		nr, err := c.Read(buf)
		if err != nil {
			return
		}

		data := buf[0:nr]
		println("Server got:", string(data))
		if strings.HasPrefix(string(data), "scale-up") {
			fmt.Println("Received scale-up event")
			appname := strings.Trim(strings.SplitN(string(data), " ", 2)[1], "\n ")
			launchGear(appname)
			WriteAppStructure()
		} else if strings.HasPrefix(string(data), "scale-down") {
			fmt.Println("Received scale-down event")
			appname := strings.Trim(strings.SplitN(string(data), " ", 2)[1], "\n ")
			removeGear(appname)
			WriteAppStructure()
		} else if strings.HasPrefix(string(data), "create-app") {
			fmt.Printf("Received create-app event: ")
			args := strings.SplitN(string(data), " ", 4)
			appname := strings.Trim(args[1], "\n ")
			image_name := strings.Trim(args[2], "\n ")
			app := Application{}
			app.Name = appname
			app.Image = image_name
			app.Fqdn = strings.Trim(args[3], "\n ")
			out, err := exec.Command("gear", "router", "create-frontend", appname, app.Fqdn).CombinedOutput()
			if err != nil {
				fmt.Printf("Error in adding new application to the router.\n")
				fmt.Println(out)
			}
			app.Gears = make([]Gear, 1)
			app.Gears = app.Gears[:0]
			GlobalAppMap[appname] = app
			WriteAppStructure()
		} else if strings.HasPrefix(string(data), "delete-app") {
			args := strings.SplitN(string(data), " ", 2)
			appname := strings.Trim(args[1], "\n ")
			app := GlobalAppMap[appname]
			if len(app.Gears) > 0 {
				fmt.Printf("Cannot delete application because it still has %s gears in it.\n", len(app.Gears))
			} else {
				delete(GlobalAppMap, appname)
				WriteAppStructure()
			}
		} else if strings.HasPrefix(string(data), "print") {
			dat, err := json.MarshalIndent(GlobalAppMap, "", "  ")
			_, err = c.Write(dat)
			if err != nil {
				fmt.Println("Write: " + err.Error())
			}
		} else if strings.HasPrefix(string(data), "help") {
			_,err = c.Write([]byte("rhc [create-app <app_name> <image_name> <fqdn>]|[delete-app <app_name>]|[scale-up <app_name>]|[scale-down <app_name>]|[print]|[help]\n"))
			if err != nil {
				fmt.Println("Write: " + err.Error())
			}
		}
	}
}

func ReadAppStructure() {
	dat, err := ioutil.ReadFile(AppData)
	if err != nil {
		GlobalAppMap = make(map[string]Application)
		return
	}
	json.Unmarshal(dat, &GlobalAppMap)
}

func WriteAppStructure() {
	dat, err := json.MarshalIndent(GlobalAppMap, "", "  ")
	if err != nil {
		fmt.Println("Failed to marshal app data - %s", err.Error())
	}
	err = ioutil.WriteFile(AppData, dat, 0644)
	if err != nil {
		fmt.Println("Failed to write to app data file - %s", err.Error())
	}
}

func removeGear(appname string) {
	// pick one gear
	app := GlobalAppMap[appname]
	l := len(app.Gears)
	if l == 0 {
		return
	}
	gear := app.Gears[l-1]
	fmt.Printf("Draining the gear %s from the router..\n", gear.Name)
	status := remove_gear_from_router(appname, gear.Name)
	if !status {
		fmt.Printf("Could not remove gear from router.")
	}
	_, err := exec.Command("gear", "delete", gear.Name).CombinedOutput()
	if err != nil {
		fmt.Printf("Error during deleting gear - %s\n", err.Error())
		return
	}
	app.Gears = app.Gears[0:(l - 1)]
	GlobalAppMap[appname] = app
	fmt.Printf("Scaled down successfully for %s.", appname)
}

func launchGear(appname string) {
	fmt.Print(GlobalAppMap)
	app := GlobalAppMap[appname]
	location := findAvailableNode()
	count := len(app.Gears)
	gear := Gear{}
	gear.Name = appname + "_" + strconv.Itoa(count)
	gear.Location = location
	fmt.Printf("Executing command - gear install %s %s/%s --start -p 8080:0\n", app.Image, location, gear.Name)
	out, err := exec.Command("gear", "install", app.Image, location+"/"+gear.Name, "--start", "-p", "8080:0").CombinedOutput()
	if err != nil {
		fmt.Printf("Error during launching new gear - ")
		fmt.Println(err.Error())
		fmt.Println(string(out))
		return
	}
	fmt.Println(string(out))
	app.Gears = append(app.Gears, gear)
	add_gear_to_router(appname, gear.Name)
	GlobalAppMap[appname] = app
}

func remove_gear_from_router(appname string, gearname string) bool {
	out, err := exec.Command("geard-router-removecontainer", "-q", "-f", appname, "-c", gearname).CombinedOutput()
	if err != nil {
		fmt.Printf("Error getting routing information from given gear.\n")
		fmt.Println(err.Error())
		fmt.Println(string(out))
		return false
	}
	fmt.Println(string(out))
	args := strings.SplitN(string(out), " ", 4)
	rout, rerr := exec.Command("gear", args[0], args[1], args[2], strings.Trim(args[3], " \n")).CombinedOutput()
	fmt.Printf("Router output : %s\n", string(rout))
	if rerr != nil {
		fmt.Println("Error while adding gear to router - %s", rerr.Error())
		return false
	}
	return true
}

func add_gear_to_router(appname string, gearname string) {
	count := 1
	fmt.Printf("geard-router-addcontainer -q -f %s -c %s\n", appname, gearname)
	out, err := exec.Command("geard-router-addcontainer", "-q", "-f", appname, "-c", gearname).CombinedOutput()
	for count < 10 && err != nil {
		count = count + 1
		fmt.Printf("Error getting routing information from new gear - ")
		fmt.Println(err.Error())
		//fmt.Println(string(out))
		fmt.Println("Retrying.")
		time.Sleep(10)
		out, err = exec.Command("geard-router-addcontainer", "-q", "-f", appname, "-c", gearname).CombinedOutput()
	}
	if err != nil {
		fmt.Println("Errored again. Giving up.")
		return
	}
	args := strings.SplitN(string(out), " ", 3)
	fmt.Printf("gear %s %s %s\n", args[0], args[1], strings.Trim(args[2], "' "))
	rout, rerr := exec.Command("gear", args[0], args[1], strings.Trim(args[2], "' \n")).CombinedOutput()
	fmt.Printf("Router output : %s\n", string(rout))
	if rerr != nil {
		fmt.Println("Error while adding gear to router - %s", rerr.Error())
		return
	}
}

func findAvailableNode() string {
	return "localhost"
}

func main() {
	filename := "/var/lib/containers/router/broker.sock"
	err := os.Remove(filename)
	if err != nil {
		// ignore
	}
	l, err := net.Listen("unix", filename)
	if err != nil {
		println("listen error", err.Error())
		return
	}

	// GlobalAppMap = make(map[string]Application)
	ReadAppStructure()
	for {
		fd, err := l.Accept()
		if err != nil {
			println("accept error", err.Error())
			return
		}

		go minister(fd)
	}
}
