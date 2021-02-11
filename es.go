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

func flattenMap(src map[string]interface{}, path [][]Node, pathIndex int, layerIndex int, mode int, header map[string]interface{}, buf *[]map[string]interface{}) {
	//fmt.Printf("pathIndex: %v\nlayerIndex: %v\n", pathIndex, layerIndex)
	//fmt.Printf("layerIndex: %v\n", layerIndex)
	keysDive := make([]string, 0)
	keysPass := make([]string, 0)
	keysCombine := make([]string, 0)
	/* construct keys slice - elements' keys of type slice and map. Other string elements goes into newHeader dict */
	for k, v := range src {
		switch sType := reflect.ValueOf(v).Type().Kind(); sType {
		case reflect.String:
			if pathIndex != 0 {
				header[path[pathIndex-mode][layerIndex].NodeName+"."+k] = v.(string)
			}
		case reflect.Float64:
			if pathIndex != 0 {
				header[path[pathIndex-mode][layerIndex].NodeName+"."+k] = v.(float64)
			}
		default:
			/* check if there are more maps inside but path is already devastated */
			if pathIndex < len(path) {
				for _, v := range path[pathIndex] {
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
		if !path[pathIndex-1][layerIndex].ToCombine {
			*buf = append(*buf, header)
		}

	} else {
		keys := make([]string, 0)
		keys = append(keysDive, keysCombine...)
		keys = append(keys, keysPass...)
		/* 		fmt.Printf("keysDive: %v\n", keysDive)
		   		fmt.Printf("keysPass: %v\n", keysPass)
		   		fmt.Printf("keys: %v, path: %v, pathIndex: %v\n", keys, path, pathIndex) */

		if pathIndex < len(path) {
			/* goes through key in keys (k). Then, for each key in original data, go for each key path[pathIndex] data (v) */
			for _, k := range keys {
				/* 				fmt.Printf("go for key: %v\n", k) */
				for i, v := range path[pathIndex] {
					switch sType := reflect.ValueOf(src[k]).Type().Kind(); sType {
					case reflect.Slice:
						if k == v.NodeName {
							src := reflect.ValueOf(src[k])
							for i := 0; i < src.Len(); i++ {
								src := src.Index(i).Interface().(map[string]interface{})
								flattenMap(src, path, pathIndex+1, i, mode, header, buf)
							}
						}
					case reflect.Map:
						if k == v.NodeName {
							src := src[k].(map[string]interface{})
							flattenMap(src, path, pathIndex+1, i, mode, header, buf)
						}
					}
				}
			}
		}
	}
}

/* func counter(src map[string]interface{}, path [][]Node, pathIndex int, layerIndex int) {
	keysPass := make([]string, 0)
	for k, v := range src {
		switch sType := reflect.ValueOf(v).Type().Kind(); sType {
		case reflect.String:
			fallthrough
		case reflect.Float64:
			fallthrough
		default:
			if pathIndex < len(path) {
				for _, v := range path[pathIndex] {
					if k == v.NodeName {
						if !v.ToDive && !v.ToCombine {
							keysPass = append(keysPass, k)
						}
					}
				}
			}
		}
	}

	if pathIndex == len(path) {
		if !path[pathIndex-1][layerIndex].ToCombine {
			fmt.Printf("exit point\n")
		}

	} else {
		if pathIndex < len(path) {
			for _, k := range keysPass {
				for i, v := range path[pathIndex] {
					switch sType := reflect.ValueOf(src[k]).Type().Kind(); sType {
					case reflect.Slice:
						if k == v.NodeName {
							src := reflect.ValueOf(src[k])
							for i := 0; i < src.Len(); i++ {
								src := src.Index(i).Interface().(map[string]interface{})
								counter(src, path, pathIndex+1, i)
							}
						}
					case reflect.Map:
						if k == v.NodeName {
							src := src[k].(map[string]interface{})
							counter(src, path, pathIndex+1, i)
						}
					}
				}
			}
		}
	}
} */

func worker(esClient *es.Client, url string, requestString string, username string, password string, path [][]Node, mode int) {
	/* 	payload := strings.NewReader(requestString)

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
	   	} */

	body := make(map[string]interface{})

	//err = json.Unmarshal([]byte(insAPIResponse.InsAPI.Outputs.Output.Body), &body)

	jsonFile, err := os.Open("rawJsonSysBgp.json")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Successfully Opened users.json")
	defer jsonFile.Close()

	bytes, _ := ioutil.ReadAll(jsonFile)

	err = json.Unmarshal(bytes, &body)

	if err != nil {
		panic(err)
	}

	var pathIndex int
	var layerIndex int
	header := make(map[string]interface{})
	buf := make([]map[string]interface{}, 0)

	flattenMap(body, path, pathIndex, layerIndex, mode, header, &buf)

	esPush(esClient, "telemetry-cadence", buf)

	//counter(body, path, pathIndex, layerIndex)
}

func main() {

	var url string = "http://10.62.130.39:8080/ins"
	var payloadString string = "{\n  \"ins_api\": {\n    \"version\": \"1.0\",\n    \"type\": \"cli_show_ascii\",\n    \"chunk\": \"0\",\n    \"sid\": \"sid\",\n    \"input\": \"show consistency-checker membership vlan 2006 detail\",\n    \"output_format\": \"json\"\n  }\n}"
	var username = "admin"
	var password = "cisco!123"

	esClient, error := esConnect("10.52.13.120", "10200")

	if error != nil {
		log.Fatalf("error: %s", error)
	}

	path := make([][]Node, 13)

	node0 := make([]Node, 1)
	node0_0 := Node{NodeName: "imdata", ToDive: false, ToCombine: false}
	node0[0] = node0_0

	node1 := make([]Node, 1)
	node1_0 := Node{NodeName: "bgpEntity", ToDive: false, ToCombine: false}
	node1[0] = node1_0

	node2 := make([]Node, 2)
	node2_0 := Node{NodeName: "attributes", ToDive: true, ToCombine: false}
	node2_1 := Node{NodeName: "children", ToDive: false, ToCombine: false}
	node2[0] = node2_0
	node2[1] = node2_1

	node3 := make([]Node, 1)
	node3_0 := Node{NodeName: "bgpInst", ToDive: false, ToCombine: false}
	node3[0] = node3_0

	node4 := make([]Node, 2)
	node4_0 := Node{NodeName: "attributes", ToDive: true, ToCombine: false}
	node4_1 := Node{NodeName: "children", ToDive: false, ToCombine: false}
	node4[0] = node4_0
	node4[1] = node4_1

	node5 := make([]Node, 1)
	node5_0 := Node{NodeName: "bgpDom", ToDive: false, ToCombine: false}
	node5[0] = node5_0

	node6 := make([]Node, 2)
	node6_0 := Node{NodeName: "attributes", ToDive: true, ToCombine: false}
	node6_1 := Node{NodeName: "children", ToDive: false, ToCombine: false}
	node6[0] = node6_0
	node6[1] = node6_1

	node7 := make([]Node, 1)
	node7_0 := Node{NodeName: "bgpPeer", ToDive: false, ToCombine: false}
	node7[0] = node7_0

	node8 := make([]Node, 2)
	node8_0 := Node{NodeName: "attributes", ToDive: true, ToCombine: false}
	node8_1 := Node{NodeName: "children", ToDive: false, ToCombine: false}
	node8[0] = node8_0
	node8[1] = node8_1

	node9 := make([]Node, 1)
	node9_0 := Node{NodeName: "bgpPeerEntry", ToDive: false, ToCombine: false}
	node9[0] = node9_0

	node10 := make([]Node, 2)
	node10_0 := Node{NodeName: "attributes", ToDive: true, ToCombine: false}
	node10_1 := Node{NodeName: "children", ToDive: false, ToCombine: false}
	node10[0] = node10_0
	node10[1] = node10_1

	node11 := make([]Node, 1)
	node11_0 := Node{NodeName: "bgpPeerAfEntry", ToDive: false, ToCombine: false}
	node11[0] = node11_0

	node12 := make([]Node, 1)
	node12_0 := Node{NodeName: "attributes", ToDive: false, ToCombine: false}
	node12[0] = node12_0

	path[0] = node0
	path[1] = node1
	path[2] = node2
	path[3] = node3
	path[4] = node4
	path[5] = node5
	path[6] = node6
	path[7] = node7
	path[8] = node8
	path[9] = node9
	path[10] = node10
	path[11] = node11
	path[12] = node12

	var mode int = 2

	worker(esClient, url, payloadString, username, password, path, mode)

	newPath := new(Path) // returns pointer to concrete data

	jsonPath, err := os.Open("mdtSysBgp.json")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Successfully Opened mdtSysBgp.json")
	defer jsonPath.Close()

	jsonPathbytes, _ := ioutil.ReadAll(jsonPath)

	err = json.Unmarshal(jsonPathbytes, newPath)

	if err != nil {
		panic(err)
	}
	fmt.Println(path)
	fmt.Println(newPath)

	for il1, vl1 := range *newPath {
		fmt.Println(il1, ":", vl1)
		for _, vl2 := range vl1.Node {
			fmt.Println(vl2.NodeName, vl2.ToCombine, vl2.ToDive)
		}
	}
}
