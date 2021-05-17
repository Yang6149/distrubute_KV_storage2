package http

import(
	"net/http"
	"fmt"
	"log"
	"distrubute_KV_storage/shardkv"
	"io/ioutil"
	"encoding/json"
)

var funcMapGet map[string]func(w http.ResponseWriter, r *http.Request,config *shardkv.Config)
var funcMapPost map[string]func(w http.ResponseWriter, r *http.Request,config *shardkv.Config)
func StartServer(config *shardkv.Config){
	
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request){
		switch r.Method{
		case "GET":
			handler,ok:= funcMapGet[r.URL.Path]
			if ok{
				handler(w,r,config)
			}
			break
		case "POST":
			handler,ok:= funcMapPost[r.URL.Path]
			if ok{
				handler(w,r,config)
			}
			break
		}
		
		
    })


    fmt.Printf("Starting server at port 8885\n")
    if err := http.ListenAndServe(":8885", nil); err != nil {
        log.Fatal(err)
    }
}

func GetAllInfo(w http.ResponseWriter, r *http.Request,config *shardkv.Config){
	fmt.Fprintf(w, config.GetAllInfoHttp())
}
func GetMasterInfo(w http.ResponseWriter, r *http.Request,config *shardkv.Config){

	var info map[string]interface{}
	body, _ := ioutil.ReadAll(r.Body)
	json.Unmarshal(body, &info)
	n ,ok:= info["n"].(int)
	if !ok{
		fmt.Println(ok)
	}
	fmt.Fprintf(w, config.GetMasterInfo(n))
}
func GetEntityInfo(w http.ResponseWriter, r *http.Request,config *shardkv.Config){
	var info map[string]int
	body, _ := ioutil.ReadAll(r.Body)
	json.Unmarshal(body, &info)
	n ,ok:= info["n"]
	if !ok{
		fmt.Println(ok)
	}
	g ,ok:= info["g"]
	if !ok{
		fmt.Println(ok)
	}
	fmt.Fprintf(w, config.GetInfo(g,n))
}

func init(){
	//bind func with path
	funcMapGet = make(map[string]func(w http.ResponseWriter, r *http.Request,config *shardkv.Config))
	funcMapPost = make(map[string]func(w http.ResponseWriter, r *http.Request,config *shardkv.Config))
	funcMapGet["/GetBasicInfo"] = GetAllInfo
	funcMapPost["/GetMasterInfo"] = GetMasterInfo
	funcMapPost["/GetEntityInfo"] = GetEntityInfo
}