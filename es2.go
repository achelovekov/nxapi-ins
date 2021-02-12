package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"reflect"

	"github.com/elastic/go-elasticsearch"
	es "github.com/elastic/go-elasticsearch"
	esapi "github.com/elastic/go-elasticsearch/esapi"
)

type insAPIGeneral struct {
	InsAPI struct {
		Outputs struct {
			Output struct {
				Body  string `json:"body"`
				Code  string `json:"code"`
				Input string `json:"input"`
				Msg   string `json:"msg"`
			} `json:"output"`
		} `json:"outputs"`
		Sid     string `json:"sid"`
		Type    string `json:"type"`
		Version string `json:"version"`
	} `json:"ins_api"`
}

type postReqHandler struct {
	esClient *es.Client
}

type Node struct {
	NodeName  string
	ToDive    bool
	ToCombine bool
}

type Path []struct {
	Node []struct {
		NodeName  string `json:"NodeName"`
		ToDive    bool   `json:"ToDive"`
		ToCombine bool   `json:"ToCombine"`
	} `json:"Node"`
}

type ESmetaData struct {
	Index struct {
		IndexName string `json:"_index"`
	} `json:"index"`
}

func PrettyPrint(src map[string]interface{}) {
	empJSON, err := json.MarshalIndent(src, "", "  ")
	if err != nil {
		log.Fatalf(err.Error())
	}
	fmt.Printf("Pretty processed output %s\n", string(empJSON))
}

func esConnect(ipaddr string, port string) (*es.Client, error) {

	var fulladdress string = "http://" + ipaddr + ":" + port

	cfg := elasticsearch.Config{
		Addresses: []string{
			fulladdress,
		},
	}

	es, _ := elasticsearch.NewClient(cfg)

	return es, nil
}

func esPush(esClient *es.Client, indexName string, buf []map[string]interface{}) []byte {

	JSONmetaData := `{"index":{"_index":"` + indexName + `"}}`

	JSONRequestData := make([]byte, 0)

	for _, v := range buf {
		JSONData, err := json.Marshal(v)
		if err != nil {
			log.Println(err)
		}

		JSONRequestData = append(JSONRequestData, JSONmetaData...)
		JSONRequestData = append(JSONRequestData, []byte("\n")...)
		JSONRequestData = append(JSONRequestData, JSONData...)
		JSONRequestData = append(JSONRequestData, []byte("\n")...)
	}

	bulkRequest := esapi.BulkRequest{
		Index: indexName,
		Body:  bytes.NewBuffer(JSONRequestData),
	}

	res, err := bulkRequest.Do(context.Background(), esClient)

	if err != nil {
		log.Fatalf("Error getting response: %s", err)
	}
	defer res.Body.Close()

	log.Println(res)

	return JSONRequestData
}

func copySlice(sli []string) []string {
	newSli := make([]string, len(sli))
	copy(newSli, sli)
	return newSli
}

func flattenMap(src map[string]interface{}, path *Path, pathIndex int, pathPassed []string, mode int, header map[string]interface{}, buf *[]map[string]interface{}) {
	keysDive := make([]string, 0)
	keysPass := make([]string, 0)
	keysCombine := make([]string, 0)
	pathPassed = copySlice(pathPassed)
	for k, v := range src {
		switch sType := reflect.ValueOf(v).Type().Kind(); sType {
		case reflect.String:
			if pathIndex != 0 {
				header[pathPassed[len(pathPassed)-mode]+"."+k] = v.(string)
			}
		case reflect.Float64:
			if pathIndex != 0 {
				header[pathPassed[len(pathPassed)-mode]+"."+k] = v.(float64)
			}
		default:
			if pathIndex < len((*path)) {
				for _, v := range (*path)[pathIndex].Node {
					if k == v.NodeName {
						if v.ToDive {
							keysDive = append(keysDive, k)
						} else if v.ToCombine {
							keysCombine = append(keysCombine, k)
						} else {
							keysPass = append(keysPass, k)
						}
					}
				}
			}
		}
	}

	if pathIndex == len(*path) {
		//fmt.Printf("pathPassed: %v\n", pathPassed)
		for _, v := range (*path)[pathIndex-1].Node {
			if pathPassed[len(pathPassed)-1] == v.NodeName && !v.ToCombine {
				//PrettyPrint(header)
				*buf = append(*buf, header) //ПРОБЛЕМА ТУТ!!!!!!!!!!БУФЕР ЗАПОЛНЯЕТСЯ ОЧЕНЬ СТРАННО!! должно быть 1 -> 1,2 -> 1,2,3, а получается 1 -> 2,2 -> 3,3,3
				//fmt.Println(*buf, "\n")
			}
		}
	} else {
		keys := make([]string, 0)
		keys = append(keysDive, keysCombine...)
		keys = append(keys, keysPass...)
		/* 		fmt.Printf("	keysDive: %v\n", keysDive)
		   		fmt.Printf("	keysPass: %v\n", keysPass)
		   		fmt.Printf("	keys: %v\n", keys) */

		if pathIndex < len((*path)) {
			for _, k := range keys {
				pathPassed = append(pathPassed, k)
				switch sType := reflect.ValueOf(src[k]).Type().Kind(); sType {
				case reflect.Map:
					//fmt.Printf("		go for Map case with key: %v\n", k)
					src := src[k].(map[string]interface{})
					flattenMap(src, path, pathIndex+1, pathPassed, mode, header, buf)
				case reflect.Slice:
					//fmt.Printf("		go for Slice case with key: %v\n", k)
					src := reflect.ValueOf(src[k])
					for i := 0; i < src.Len(); i++ {
						src := src.Index(i).Interface().(map[string]interface{})
						flattenMap(src, path, pathIndex+1, pathPassed, mode, header, buf)
					}
				}
			}
		}
	}
}

func worker(esClient *es.Client, path *Path, mode int) {

	body := make(map[string]interface{})
	jsonFile, err := os.Open("rawJsonSysBgp.json")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Successfully Opened rawJsonSysBgp.json")
	defer jsonFile.Close()

	bytes, _ := ioutil.ReadAll(jsonFile)

	err = json.Unmarshal(bytes, &body)

	if err != nil {
		panic(err)
	}

	var pathIndex int
	header := make(map[string]interface{})
	buf := make([]map[string]interface{}, 0)
	pathPassed := make([]string, 0)

	flattenMap(body, path, pathIndex, pathPassed, mode, header, &buf)

	for _, v := range buf {
		fmt.Println(v)
	}
	//esPush(esClient, "telemetry-cadence", buf)
}

func main() {
	esClient, error := esConnect("10.52.13.120", "10200")
	if error != nil {
		log.Fatalf("error: %s", error)
	}

	jsonPath, err := os.Open("mdtSysBgp.json")
	if err != nil {
		fmt.Println(err)
	}

	path := new(Path) // returns pointer to concrete data
	defer jsonPath.Close()

	jsonPathBytes, _ := ioutil.ReadAll(jsonPath)

	err = json.Unmarshal(jsonPathBytes, path)
	if err != nil {
		panic(err)
	}

	var mode int = 2
	worker(esClient, path, mode)
}
