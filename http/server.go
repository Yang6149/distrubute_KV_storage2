package http

import (
	"distrubute_KV_storage/shardkv"
	"distrubute_KV_storage/shardmaster"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type Config struct {
	IsMaster bool
	Master   *shardmaster.ShardMaster
	Slaver   *shardkv.ShardKV
}

var funcMapGet map[string]func(w http.ResponseWriter, r *http.Request, config *Config)
var funcMapPost map[string]func(w http.ResponseWriter, r *http.Request, config *Config)

func StartServer(config *Config) {

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			handler, ok := funcMapGet[r.URL.Path]
			if ok {
				handler(w, r, config)
			}
			break
		case "POST":
			handler, ok := funcMapPost[r.URL.Path]
			if ok {
				handler(w, r, config)
			}
			break
		}

	})

	fmt.Printf("Starting server at port 8885\n")
	if err := http.ListenAndServe(":8885", nil); err != nil {
		log.Fatal(err)
	}
}

type LInfo struct {
	I int
	B bool
}

func GetAllInfo(w http.ResponseWriter, r *http.Request, config *Config) {
	i := 0
	b := false

	if config.IsMaster {
		i, b = config.Master.Raft().GetState()
	} else {
		i, b = config.Slaver.Raft().GetState()
	}
	L := LInfo{}
	L.I = i
	L.B = b
	jsons, errs := json.Marshal(L) //转换成JSON返回的是byte[]
	if errs != nil {
		fmt.Println(errs.Error())
	}
	fmt.Fprintf(w, string(jsons))
}

type masterInfo struct {
	Config shardmaster.Config
}

func GetMasterInfo(w http.ResponseWriter, r *http.Request, config *Config) {
	master := config.Master
	info := masterInfo{}
	info.Config = master.GetNewConfig()
	jsons, errs := json.Marshal(info) //转换成JSON返回的是byte[]
	if errs != nil {
		fmt.Println(errs.Error())
	}
	fmt.Fprintf(w, string(jsons))
}

type entityInfo struct {
	// Config shardmaster.Config
	ShardNum int
	KeyNum   int
	IndexLen int
}

func GetEntityInfo(w http.ResponseWriter, r *http.Request, config *Config) {
	kv := config.Slaver
	info := entityInfo{}
	info.IndexLen = kv.Raft().GetlogLen()
	info.ShardNum = kv.GetShardNum()
	info.KeyNum = kv.GetKeyNum()
	jsons, errs := json.Marshal(info) //转换成JSON返回的是byte[]
	if errs != nil {
		fmt.Println(errs.Error())
	}
	fmt.Fprintf(w, string(jsons))
}

func init() {
	//bind func with path
	funcMapGet = make(map[string]func(w http.ResponseWriter, r *http.Request, config *Config))
	funcMapPost = make(map[string]func(w http.ResponseWriter, r *http.Request, config *Config))
	funcMapGet["/GetBasicInfo"] = GetAllInfo
	funcMapGet["/GetMasterInfo"] = GetMasterInfo
	funcMapGet["/GetEntityInfo"] = GetEntityInfo
}
