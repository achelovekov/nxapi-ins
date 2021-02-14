package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"reflect"
	"strings"

	es "github.com/elastic/go-elasticsearch"
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
	NodeName string
	ToDive   bool
}

func PrettyPrint(src map[string]interface{}) {
	empJSON, err := json.MarshalIndent(src, "", "  ")
	if err != nil {
		log.Fatalf(err.Error())
	}
	fmt.Printf("Pretty processed output %s\n", string(empJSON))
}

func flattenMap(src map[string]interface{}, path [][]Node, pathIndex int, layerIndex int, header map[string]interface{}) {

	keysDive := make([]string, 0)
	keysPass := make([]string, 0)
	/* construct keys slice - elements' keys of type slice and map. Other string elements goes into newHeader dict */
	for k, v := range src {
		switch sType := reflect.ValueOf(v).Type().Kind(); sType {
		case reflect.String:
			if pathIndex != 0 {
				header[path[pathIndex-1][layerIndex].NodeName+"."+k] = v.(string)
			}
		case reflect.Float64:
			if pathIndex != 0 {
				header[path[pathIndex-1][layerIndex].NodeName+"."+k] = v.(float64)
			}
		default:
			for _, v := range path[pathIndex] {
				if k == v.NodeName {
					if v.ToDive {
						keysDive = append(keysDive, k)
					} else {
						keysPass = append(keysPass, k)
					}
				}
			}
		}
	}

	keys := make([]string, 0)
	keys = append(keysDive, keysPass...)
	/* 	fmt.Printf("keysDive: %v\n", keysDive)
	   	fmt.Printf("keysPass: %v\n", keysPass)
	   	fmt.Printf("keys: %v, path: %v, pathIndex: %v\n", keys, path, pathIndex) */

	if pathIndex < len(path) {
		/* goes through key in keys (k). Then, for each key in original data, go for each key path[pathIndex] data (v) */
		for _, k := range keys {
			/* 			fmt.Printf("go for key: %v\n", k) */
			for i, v := range path[pathIndex] {
				switch sType := reflect.ValueOf(src[k]).Type().Kind(); sType {
				case reflect.Slice:
					if k == v.NodeName {
						src := reflect.ValueOf(src[k])
						for i := 0; i < src.Len(); i++ {
							src := src.Index(i).Interface().(map[string]interface{})
							flattenMap(src, path, pathIndex+1, i, header)
						}
					}
				case reflect.Map:
					if k == v.NodeName {
						src := src[k].(map[string]interface{})
						flattenMap(src, path, pathIndex+1, i, header)
					}
				}
			}
		}
	} else {
		PrettyPrint(header)
	}
}

func worker(url string, requestString string, username string, password string, path [][]Node) {
	payload := strings.NewReader(requestString)

	req, _ := http.NewRequest("POST", url, payload)

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Cache-Control", "no-cache")
	req.SetBasicAuth(username, password)

	res, _ := http.DefaultClient.Do(req)

	defer res.Body.Close()

	responseBody, _ := ioutil.ReadAll(res.Body)

	var insAPIResponse insAPIGeneral
	err := json.Unmarshal(responseBody, &insAPIResponse)
	if err != nil {
		panic(err)
	}

	body := make(map[string]interface{})
	err = json.Unmarshal([]byte(insAPIResponse.InsAPI.Outputs.Output.Body), &body)

	if err != nil {
		panic(err)
	}

	var pathIndex int
	var layerIndex int
	header := make(map[string]interface{})
	flattenMap(body, path, pathIndex, layerIndex, header)
}

func main() {

	var url string = "http://10.62.130.39:8080/ins"
	var payloadString string = "{\n  \"ins_api\": {\n    \"version\": \"1.0\",\n    \"type\": \"cli_show_ascii\",\n    \"chunk\": \"0\",\n    \"sid\": \"sid\",\n    \"input\": \"show consistency-checker membership port-channels detail\",\n    \"output_format\": \"json\"\n  }\n}"
	var username = "admin"
	var password = "cisco!123"

}
