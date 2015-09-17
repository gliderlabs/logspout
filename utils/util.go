package utils

import (
        "io/ioutil"
        "encoding/json"
	"time"
	"net"
	"errors"
)

var UUID string
var M1 map[string]string

type Message struct{
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
        buff, err := ioutil.ReadFile("/etc/omega/agent/omega-agent.conf")

        if err != nil {
                panic("open file failed")
        }

        var m map[string]interface{}
        json.Unmarshal(buff, &m)

        if str, ok := m["OmegaUUID"].(string); ok {
                UUID = str
        }
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
	ip, _ := GetIp()
	data, err := HttpGet("http://" + ip + ":5051/slave(1)/state.json")
	if err == nil {
		mg := make(map[string]string)
		mg["omega-slave"] = "omega-slave omega-slave"
		mg["omega-marathon"] = "omega-marathon omega-marathon"
		mg["omega-master"] = "omega-master omega-master"
		mg["omega-zookeeper"] = "omega-zookeeper omega-zookeeper"
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
