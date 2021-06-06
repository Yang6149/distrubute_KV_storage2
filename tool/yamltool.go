package tool

import (
	"io/ioutil"
	"log"
	"strconv"

	"gopkg.in/yaml.v2"
)

type Conf struct {
	Ip             string   `yaml:"ip"` //yaml：yaml格式 enabled：属性的为enabled
	Port           int      `yaml:"port"`
	RaftPort       int      `yaml:"raftPort"`
	IsMaster       bool     `yaml:"ismaster"`
	MasterN        int      `yaml:"masterN"`
	GroupN         int      `yaml:"GroupN"`
	Num            int      `yaml:"Num"`
	MasterIP       []string `yaml:"masterIP"`
	Spec           Spec     `yaml: "spec"`
	Id             int      `yaml: "id"`
	Group          int      `yaml: "group"`
	IsMasterClient bool     `yaml: "ismasterclient"`
	IsClient       bool     `yaml: "isclient"`
}

type Groups struct {
	Id int      `yaml:"id"`
	IP []string `yaml:"ip"`
}
type Spec struct {
	Conditions []Conditions `yaml: "conditions"`
}

type Conditions struct {
	Id string   `yaml:"id"`
	IP []string `yaml: "ip"`
}

func (c *Conf) GetConf() *Conf {
	yamlFile, err := ioutil.ReadFile("conf.yaml")
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}

	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}
	return c
}
func (c *Conf) GetMyRaftIp() string {
	return c.Ip + ":" + strconv.Itoa(c.RaftPort)
}
func (c *Conf) GetMyServeIp() string {
	return c.Ip + ":" + strconv.Itoa(c.Port)
}
