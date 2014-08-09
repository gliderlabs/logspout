package main

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"log"

	"github.com/CMGS/consistent"
)

type AttachEvent struct {
	Type string
	ID   string
	Name string
}

type Log struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Data    string `json:"data"`
	Appname string `json:"appname"`
	Tag     string `json:"tag"`
	Port    string `json:"port"`
}

type Route struct {
	ID       string  `json:"id"`
	Source   *Source `json:"source,omitempty"`
	Target   *Target `json:"target"`
	backends *consistent.Consistent
	closer   chan bool
}

func (s *Route) loadBackends() {
	s.backends = consistent.New()
	for _, addr := range s.Target.Addrs {
		s.backends.Add(addr)
	}
}

type Source struct {
	ID     string   `json:"id,omitempty"`
	Name   string   `json:"name,omitempty"`
	Filter string   `json:"filter,omitempty"`
	Types  []string `json:"types,omitempty"`
}

func (s *Source) All() bool {
	return s.ID == "" && s.Name == "" && s.Filter == ""
}

type Target struct {
	Addrs     []string `json:"addrs"`
	AppendTag string   `json:"append_tag,omitempty"`
}

func marshal(obj interface{}) []byte {
	bytes, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		log.Println("marshal:", err)
	}
	return bytes
}

func unmarshal(input io.ReadCloser, obj interface{}) error {
	body, err := ioutil.ReadAll(input)
	if err != nil {
		return err
	}
	err = json.Unmarshal(body, obj)
	if err != nil {
		return err
	}
	return nil
}
