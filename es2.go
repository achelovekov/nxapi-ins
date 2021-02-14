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

type Config struct {
	ESHost       string `json:"ESHost"`
	ESPort       string `json:"ESPort"`
	MDTPathsFile string `json:"MDTPathsFile"`
	ESIndex      string `json:"ESIndex"`
}

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

type MDTPathDefinitions []struct {
	MdtPath     string `json:"mdtPath"`
	MdtPathFile string `json:"mdtPathFile"`
}

type MDTPaths map[string]Path

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

func copyMap(ma map[string]interface{}) map[string]interface{} {
	newMap := make(map[string]interface{})
	for k, v := range ma {
		newMap[k] = v
	}
	return newMap
}

func flattenMap(src map[string]interface{}, path Path, pathIndex int, pathPassed []string, mode int, header map[string]interface{}, buf *[]map[string]interface{}) {
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
			if pathIndex < len(path) {
				for _, v := range path[pathIndex].Node {
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

	if pathIndex == len(path) {
		for _, v := range path[pathIndex-1].Node {
			if pathPassed[len(pathPassed)-1] == v.NodeName && !v.ToCombine {
				newHeader := copyMap(header)
				*buf = append(*buf, newHeader)
			}
		}
	} else {
		keys := make([]string, 0)
		keys = append(keysDive, keysCombine...)
		keys = append(keys, keysPass...)

		if pathIndex < len(path) {
			for _, k := range keys {
				pathPassed = append(pathPassed, k)
				switch sType := reflect.ValueOf(src[k]).Type().Kind(); sType {
				case reflect.Map:
					src := src[k].(map[string]interface{})
					flattenMap(src, path, pathIndex+1, pathPassed, mode, header, buf)
				case reflect.Slice:
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

func worker(esClient *es.Client, indexName string, path Path, mode int) {
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
	esPush(esClient, indexName, buf)
}

func LoadMDTPaths(fileName string) MDTPaths {

	var MDTPathDefinitions MDTPathDefinitions
	MDTPaths := make(MDTPaths)

	MDTPathDefinitionsFile, err := os.Open(fileName)
	if err != nil {
		fmt.Println(err)
	}
	defer MDTPathDefinitionsFile.Close()

	MDTPathDefinitionsFileBytes, err := ioutil.ReadAll(MDTPathDefinitionsFile)
	if err != nil {
		fmt.Println(err)
	}

	err = json.Unmarshal(MDTPathDefinitionsFileBytes, &MDTPathDefinitions)
	if err != nil {
		fmt.Println(err)
	}

	for _, v := range MDTPathDefinitions {
		var path Path

		pathFile, err := os.Open(v.MdtPathFile)
		if err != nil {
			fmt.Println(err)
		}
		defer pathFile.Close()

		pathFileBytes, _ := ioutil.ReadAll(pathFile)

		err = json.Unmarshal(pathFileBytes, &path)
		if err != nil {
			fmt.Println(err)
		}
		MDTPaths[v.MdtPath] = path
	}
	return MDTPaths
}

func main() {

	var Config Config
	var MDTPaths MDTPaths
	var mode int = 2 // mode 2 for cadence

	ConfigFile, err := os.Open("config.json")
	if err != nil {
		fmt.Println(err)
	}
	defer ConfigFile.Close()

	ConfigFileBytes, _ := ioutil.ReadAll(ConfigFile)

	err = json.Unmarshal(ConfigFileBytes, &Config)
	if err != nil {
		fmt.Println(err)
	}

	esClient, error := esConnect(Config.ESHost, Config.ESPort)
	if error != nil {
		log.Fatalf("error: %s", error)
	}

	MDTPaths = LoadMDTPaths(Config.MDTPathsFile)
	worker(esClient, Config.ESIndex, MDTPaths["sys/bgp"], mode)
}
