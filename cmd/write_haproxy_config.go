package main

import (
	"fmt"
	"io/ioutil"
	//"encoding/json"
	"github.com/openshift/geard/router"
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
)

func writeServer(f *os.File, id string, s *router.Endpoint) {
	f.WriteString(fmt.Sprintf("  server %s %s:%s check inter 5000ms\n", id, s.IP, s.Port))
}

func main() {
	router.ReadRoutes()
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
	for frontendname, frontend := range router.GlobalRoutes {
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
