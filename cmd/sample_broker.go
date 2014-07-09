
package main

import (
	"fmt"
	"os/exec"
	"net"
	"strings"
	"strconv"
	"time"
	"os"
)

type Gear struct {
	Name		string
	Location	string
}

type Application struct {
	Name	string
	Fqdn    string
	Image	string
	Gears	[]Gear
}

var GlobalAppMap map[string]Application 

func minister(c net.Conn) {
	for {
		buf := make([]byte, 1024)
		nr, err := c.Read(buf)
		if err != nil {
			return
		}

		data := buf[0:nr]
		println("Server got:", string(data))
		_, err = c.Write(data)
		if err != nil {
			panic("Write: " + err.Error())
		}
		if strings.HasPrefix(string(data), "scale-up") {
			fmt.Println("Received scale-up event")
			appname := strings.Trim(strings.SplitN(string(data)," ", 2)[1], "\n ")
			launchGear(appname)
		} else if strings.HasPrefix(string(data), "scale-down") {
			fmt.Println("Received scale-down event")
			appname := strings.Trim(strings.SplitN(string(data)," ", 2)[1], "\n ")
			removeGear(appname)
		} else if strings.HasPrefix(string(data), "create-app") {
			fmt.Printf("Received create-app event: ")
			args := strings.SplitN(string(data)," ", 4)
			appname := strings.Trim(args[1], "\n ")
			image_name := strings.Trim(args[2], "\n ")
			app := Application{}
			app.Name =  appname
			app.Image = image_name
			app.Fqdn = strings.Trim(args[3], "\n ")
			out,err := exec.Command("gear", "router", "create-frontend", appname, app.Fqdn).CombinedOutput()
			if err!=nil {
				fmt.Printf("Error in adding new application to the router.\n")
				fmt.Println(out)
			}
			app.Gears = make([]Gear,1)
			GlobalAppMap[appname] = app
		}
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
	_,err := exec.Command("gear", "delete", gear.Name).CombinedOutput()
	app.Gears = app.Gears[0:(l-2)]
	if err!=nil {
		fmt.Printf("Error during deleting gear - %s\n", err.Error())
		return
	}
}

func launchGear(appname string) {
	fmt.Print(GlobalAppMap)
	app := GlobalAppMap[appname]
	location := findAvailableNode()
	count := len(app.Gears)
	gear := Gear{}
	gear.Name = appname+"_"+strconv.Itoa(count)
	gear.Location = location
	fmt.Printf("Executing command - gear install %s %s/%s --start -p 8080:0\n", app.Image, location, gear.Name)
	out, err := exec.Command("gear", "install", app.Image, location+"/"+gear.Name, "--start", "-p", "8080:0").CombinedOutput()
	if err!=nil {
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

func add_gear_to_router(appname string, gearname string) {
	count := 1
	fmt.Printf("geard-router-addcontainer -q -f %s -c %s\n", appname, gearname)
	out, err := exec.Command("geard-router-addcontainer", "-q", "-f", appname, "-c", gearname).CombinedOutput()
	for (count<5 && err!=nil) {
		count = count +1
		fmt.Printf("Error getting routing information from new gear - ")
		fmt.Println(err.Error())
		fmt.Println(string(out))
		fmt.Println("Retrying.")
		time.Sleep(10)
		out, err = exec.Command("geard-router-addcontainer", "-q", "-f", appname, "-c", gearname).CombinedOutput()
	}
	if err!=nil {
		fmt.Println("Errored again. Giving up.")
		return
	}
	args := strings.SplitN(string(out)," ", 3)
	fmt.Printf("gear %s %s %s\n", args[0], args[1], strings.Trim(args[2], "' "))
	rout, rerr := exec.Command("gear", args[0], args[1], strings.Trim(args[2], "' \n")).CombinedOutput()
	fmt.Printf("Router output : %s\n", string(rout))
	if rerr!=nil {
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

	GlobalAppMap = make(map[string]Application)
	for {
		fd, err := l.Accept()
		if err != nil {
			println("accept error", err.Error())
			return
		}

		go minister(fd)
	}
}
