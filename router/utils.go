package router

import (
        "io/ioutil"
        "encoding/json"
)

var UUID string

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

