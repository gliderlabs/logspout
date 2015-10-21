package utils

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	_ "io/ioutil"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

var UUID string
var M1 map[string]string
var IP string
var Hostname string
var UserId string
var ClusterId string

type Message struct {
	FrameWorks []struct {
		Executors []struct {
			Container string `json:"container"`
			Tasks     []struct {
				SlaveId string `json:"slave_id"`
				State   string `json:"state"`
				Name    string `json:"name"`
				Id      string `json:"id"`
			} `json:"tasks"`
		} `json:"executors"`
	} `json:"frameworks"`
	HostName string `json:"hostname"`
}

func init() {
	UUID = os.Getenv("HOST_ID")
	if UUID == "" {
		log.Println("cat't found uuid")
		os.Exit(0)
	}
	UserId = os.Getenv("USER_ID")
	if UserId == "" {
		log.Println("cat't found userid")
		os.Exit(0)
	}
	ClusterId = os.Getenv("CLUSTER_ID")
	if ClusterId == "" {
		log.Println("cat't found clusterid")
		os.Exit(0)
	}
	Hostname, _ = os.Hostname()
	IP, _ = GetIp()
	M1 = getCnames()
}

func Run() {
	timer := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-timer.C:
			GetMesosInfo()
		}
	}
}

func GetMesosInfo() {
	data, err := HttpGet("http://" + IP + ":5051/slave(1)/state.json")
	if err == nil {
		mg := getCnames()
		var m Message
		json.Unmarshal([]byte(data), &m)
		if len(m.FrameWorks) > 0 {
			for _, fw := range m.FrameWorks {
				if len(fw.Executors) > 0 {
					for _, ex := range fw.Executors {
						if len(ex.Tasks) > 0 {
							for _, ts := range ex.Tasks {
								mcn := "/mesos-" + ts.SlaveId + "." + ex.Container
								mg[mcn] = ts.Name + " " + ts.Id
							}
						}
					}
				}
			}
		}
		M1 = mg
	}
}

func GetIp() (ip string, err error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			return ip.String(), nil
		}
	}
	return "", errors.New("are you connected to the network?")
}

func ConnTCP(address string) (net.Conn, error) {
	raddr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return nil, err
	}
	conn, err := net.DialTCP("tcp", nil, raddr)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func ConnTLS(address string) (net.Conn, error) {
	cert, err := tls.LoadX509KeyPair("/root/ssl/client.pem", "/root/ssl/client.key")
	if err != nil {
		return nil, err
	}
	config := tls.Config{Certificates: []tls.Certificate{cert}, InsecureSkipVerify: true}
	conn, err := tls.Dial("tcp", address, &config)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func getCnames() map[string]string {
	cnames := os.Getenv("CNAMES")
	if cnames != "" {
		cmap := make(map[string]string)
		ca := strings.Split(cnames, ",")
		for _, cname := range ca {
			rname := strings.Replace(cname, "/", "", 1)
			cmap[cname] = rname + " " + rname
		}
		return cmap
	}
	return nil
}
