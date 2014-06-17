package main

import (
	"fmt"
	"encoding/json"
	"io/ioutil"
	"os"
)

const (
	ProtocolHttp   = "http"
	ProtocolHttps  = "https"
	ProtocolTls    = "tls"
	ConfigTemplate = "/var/lib/haproxy/conf/haproxy_template.conf"
	ConfigFile     = "/var/lib/haproxy/conf/haproxy.config"
	HostMapFile    = "/var/lib/haproxy/conf/host_be.map"
	HostMapSniFile    = "/var/lib/haproxy/conf/host_be_sni.map"
	HostMapResslFile    = "/var/lib/haproxy/conf/host_be_ressl.map"
	HostMapWsFile    = "/var/lib/haproxy/conf/host_be_ws.map"
	RouteFile    = "/var/lib/containers/router/routes.json"
)

type Frontend struct {
	Name        string
	HostAliases []string
	BeTable     map[string]Backend
	EndpointTable map[string]Endpoint
}

type Backend struct {
	Id           string
	FePath       string
	BePath       string
	Protocols    []string
	EndpointIds    []string
	SslTerm      string
	Certificates []Certificate
}

type Certificate struct {
	Id                 string
	Contents           []byte
	PrivateKey         []byte
	PrivateKeyPassword string
}

type Endpoint struct {
	Id   string
	IP   string
	Port string
}

var GlobalRoutes map[string]Frontend

func ReadRoutes() {
	//fmt.Printf("Reading routes file (%s)\n", RouteFile)
	dat, err := ioutil.ReadFile(RouteFile)
	if err != nil {
		GlobalRoutes = make(map[string]Frontend)
		return
	}
	json.Unmarshal(dat, &GlobalRoutes)
}

func writeServer(f *os.File, id string, s *Endpoint) {
	f.WriteString(fmt.Sprintf("  server %s %s:%s check inter 5000ms\n", id, s.IP, s.Port))
}

func main() {
	ReadRoutes()
	hf, herr := os.Create(HostMapFile)
	if herr != nil {
		fmt.Println("Error creating host map file - %s", herr.Error())
		os.Exit(1)
	}
	dat, terr := ioutil.ReadFile(ConfigTemplate)
	if terr != nil {
		fmt.Println("Error reading from template configuration - %s", terr.Error())
		os.Exit(1)
	}
	f, err := os.Create(ConfigFile)
	if err != nil {
		fmt.Println("Error opening file haproxy.conf - %s", err.Error())
		os.Exit(1)
	}
	f.WriteString(string(dat))
	for frontendname, frontend := range GlobalRoutes {
		for host := range frontend.HostAliases {
			if frontend.HostAliases[host] != "" {
				hf.WriteString(fmt.Sprintf("%s %s\n", frontend.HostAliases[host], frontendname))
			}
		}

		f.WriteString(fmt.Sprintf("backend be_%s\n  mode http\n  balance leastconn\n  timeout check 5000ms\n", frontendname))
		for seid, se := range frontend.EndpointTable {
			writeServer(f, seid, &se)
		}
		f.WriteString("\n")
	}
	f.Close()
}
