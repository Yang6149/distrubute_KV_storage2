package http

import (
	"distrubute_KV_storage/tool"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

var timeout = time.Duration(2 * time.Second)

func dialTimeout(network, addr string) (net.Conn, error) {
	return net.DialTimeout(network, addr, timeout)
}

var transport = http.Transport{
	Dial: dialTimeout,
}
var client = http.Client{
	Transport: &transport,
}

var ClifuncMapGet map[string]func(w http.ResponseWriter, r *http.Request)
var ClifuncMapPost map[string]func(w http.ResponseWriter, r *http.Request)
var c tool.Conf

func StartServer2() {

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			handler, ok := ClifuncMapGet[r.URL.Path]
			if ok {
				handler(w, r)
			}
			break
		case "POST":
			handler, ok := ClifuncMapPost[r.URL.Path]
			if ok {
				handler(w, r)
			}
			break
		}

	})

	fmt.Printf("Starting server at port 8885\n")
	if err := http.ListenAndServe(":8885", nil); err != nil {
		log.Fatal(err)
	}
}

type basicInfo struct {
	MasterNum       int
	GroupNum        int
	Num             int
	IsMasterLeader  map[int]bool
	IsLeader        map[int]map[int]bool
	IsMasterConnect map[int]bool
	IsConnect       map[int]map[int]bool
	MasterTerm      map[int]int
	Term            map[int]map[int]int
}

func GetAllInfo2(w http.ResponseWriter, r *http.Request) {
	res1 := basicInfo{
		MasterNum:       c.MasterN,
		GroupNum:        c.GroupN,
		Num:             c.Num,
		IsMasterLeader:  make(map[int]bool),
		IsLeader:        make(map[int]map[int]bool),
		IsMasterConnect: make(map[int]bool),
		IsConnect:       make(map[int]map[int]bool),
		MasterTerm:      make(map[int]int),
		Term:            make(map[int]map[int]int),
	}
	masters := make([]string, 0)
	slaver := make([][]string, 0)
	for i := range c.MasterIP {
		masters = append(masters, c.MasterIP[i])
	}
	for i := range c.Spec.Conditions {
		s := make([]string, 0)
		for j := range c.Spec.Conditions[i].IP {
			s = append(s, c.Spec.Conditions[i].IP[j])
		}
		slaver = append(slaver, s)
	}
	for i := range masters {
		ips := masters[i]
		strs := strings.Split(ips, " ")
		resp, err := client.Get("http://" + strs[0] + ":8885" + "/GetBasicInfo")
		res1.IsMasterConnect[i] = true
		if err != nil {
			res1.IsMasterConnect[i] = false
			// fmt.Println(err)
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		l := LInfo{}
		err = json.Unmarshal(body, &l)
		if err != nil {
			fmt.Println("unmarshal err !22azxc")
		}
		res1.IsMasterLeader[i] = l.B
		res1.MasterTerm[i] = l.I
	}
	for i := range slaver {
		res1.IsConnect[i] = make(map[int]bool)
		res1.IsLeader[i] = make(map[int]bool)
		res1.Term[i] = make(map[int]int)
		for j := range slaver[i] {
			ips := slaver[i][j]
			strs := strings.Split(ips, " ")
			resp, err := client.Get("http://" + strs[0] + ":8885" + "/GetBasicInfo")
			res1.IsConnect[i][j] = true
			if err != nil {
				// fmt.Println("123123213", err)
				res1.IsConnect[i][j] = false
				continue
			}
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			l := LInfo{}
			err = json.Unmarshal(body, &l)
			if err != nil {
				fmt.Println("unmarshal err !22azxc")
			}
			res1.IsLeader[i][j] = l.B
			res1.Term[i][j] = l.I
		}
	}
	jsons, errs := json.Marshal(res1) //转换成JSON返回的是byte[]
	if errs != nil {
		fmt.Println(errs.Error())
	}
	fmt.Fprintf(w, string(jsons))
}
func GetMasterInfo2(w http.ResponseWriter, r *http.Request) {

	var info map[string]interface{}
	body, _ := ioutil.ReadAll(r.Body)
	json.Unmarshal(body, &info)
	n, ok := info["n"].(int)
	if !ok {
		fmt.Println(ok)
	}
	masters := make([]string, 0)
	for i := range c.MasterIP {
		masters = append(masters, c.MasterIP[i])
	}
	ips := masters[n]
	strs := strings.Split(ips, " ")
	resp, err := client.Get("http://" + strs[0] + ":8885" + "/GetMasterInfo")
	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("afogaf", err)
	}
	fmt.Fprintf(w, string(body))
}
func GetEntityInfo2(w http.ResponseWriter, r *http.Request) {
	slaver := make([][]string, 0)
	for i := range c.Spec.Conditions {
		s := make([]string, 0)
		for j := range c.Spec.Conditions[i].IP {
			s = append(s, c.Spec.Conditions[i].IP[j])
		}
		slaver = append(slaver, s)
	}
	var info map[string]int
	body, _ := ioutil.ReadAll(r.Body)
	json.Unmarshal(body, &info)
	n, ok := info["n"]
	if !ok {
		fmt.Println("mei n")
	}
	g, ok := info["g"]
	if !ok {
		fmt.Println("mei g")
	}
	ips := slaver[g][n]
	strs := strings.Split(ips, " ")
	resp, err := client.Get("http://" + strs[0] + ":8885" + "/GetEntityInfo")
	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("afdoaf", err)
	}
	fmt.Fprintf(w, string(body))

	// fmt.Fprintf(w, config.GetInfo(g, n))
}

func init() {
	//bind func with path
	ClifuncMapGet = make(map[string]func(w http.ResponseWriter, r *http.Request))
	ClifuncMapPost = make(map[string]func(w http.ResponseWriter, r *http.Request))
	ClifuncMapGet["/GetBasicInfo"] = GetAllInfo2
	ClifuncMapPost["/GetMasterInfo"] = GetMasterInfo2
	ClifuncMapPost["/GetEntityInfo"] = GetEntityInfo2
	c.GetConf()
}
