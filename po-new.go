package main

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"

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
	NodeName  string
	ToDive    bool
	ToCombine bool
}

func PrettyPrint(src map[string]interface{}) {
	empJSON, err := json.MarshalIndent(src, "", "  ")
	if err != nil {
		log.Fatalf(err.Error())
	}
	fmt.Printf("Pretty processed output %s\n", string(empJSON))
}

func flattenMap(src map[string]interface{}, path [][]Node, pathIndex int, layerIndex int, header map[string]interface{}) {
	//fmt.Printf("layerIndex: %v\n", layerIndex)
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
			/* check if there are more maps inside but path is already devastated */
			if pathIndex < len(path) {
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
	}

	keys := make([]string, 0)
	keys = append(keysDive, keysPass...)
	/* 	fmt.Printf("keysDive: %v\n", keysDive)
	   	fmt.Printf("keysPass: %v\n", keysPass)
	   	fmt.Printf("keys: %v, path: %v, pathIndex: %v\n", keys, path, pathIndex) */

	if pathIndex < len(path) {
		/* goes through key in keys (k). Then, for each key in original data, go for each key path[pathIndex] data (v) */
		for _, k := range keys {
			//fmt.Printf("go for key: %v\n", k)
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
		if !path[pathIndex-1][layerIndex].ToCombine {
			PrettyPrint(header)
		}
	}
}

func worker(url string, requestString string, username string, password string, path [][]Node) {
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

	raw := `{
		"result": {
		  "status": "CC_STATUS_OK",
		  "checkers": [
			{
			  "version": 1,
			  "type": "CC_TYPE_IF_AGGR_MEMBERSHIP",
			  "status": "CC_STATUS_OK",
			  "platformDetails": {
				"classType": "CC_PLTFM_NXOS_CSC_ROCKY"
			  },
			  "recoveryActions": [],
			  "failedEntities": [],
			  "passedEntities": [
				{
				  "key": [
					{
					  "type": "CC_ENTITY_IF_AGGR",
					  "value": "po20"
					}
				  ],
				  "value": {
					"errorCode": "CC_ERR_SUCCESS",
					"entityDetails": {
					  "ifIndex": 369098771,
					  "nxIndex": 4768,
					  "nsPId": 0,
					  "dMod": 0,
					  "dPId": 2,
					  "LTL": 3,
					  "srcId": 0
					},
					"recoveryActions": [],
					"checkedProperties": {
					  "tahoe": {
						"egrNumPaths_Sw_Hw": {
						  "errorCode": "CC_ERR_SUCCESS",
						  "expected": 1,
						  "actual": 1,
						  "expectedDetails": {},
						  "actualDetails": {
							"hwTableName": "tah_hom_luc_ucportchannelconfigtable 1538 changed",
							"hwTableField": "num_paths, base_ptr"
						  }
						},
						"egrMembers_Sw_Hw": {
						  "errorCode": "CC_ERR_SUCCESS",
						  "expected": [
							"eth1/20"
						  ],
						  "actual": [
							"eth1/20"
						  ],
						  "expectedDetails": {},
						  "actualDetails": {
							"hwTableName": "tah_hom_luc_ucportchannelmembertable 1537",
							"hwTableField": "dst_chip, dst_port"
						  }
						}
					  }
					}
				  }
				},
				{
				  "key": [
					{
					  "type": "CC_ENTITY_IF_AGGR",
					  "value": "po3967"
					}
				  ],
				  "value": {
					"errorCode": "CC_ERR_SUCCESS",
					"entityDetails": {
					  "ifIndex": 369102718,
					  "nxIndex": 4768,
					  "nsPId": 0,
					  "dMod": 0,
					  "dPId": 3,
					  "LTL": 4,
					  "srcId": 0
					},
					"recoveryActions": [],
					"checkedProperties": {
					  "tahoe": {
						"egrNumPaths_Sw_Hw": {
						  "errorCode": "CC_ERR_SUCCESS",
						  "expected": 2,
						  "actual": 2,
						  "expectedDetails": {},
						  "actualDetails": {
							"hwTableName": "tah_hom_luc_ucportchannelconfigtable 1539 changed",
							"hwTableField": "num_paths, base_ptr"
						  }
						},
						"egrMembers_Sw_Hw": {
						  "errorCode": "CC_ERR_SUCCESS",
						  "expected": [
							"eth1/53",
							"eth1/54"
						  ],
						  "actual": [
							"eth1/53",
							"eth1/54"
						  ],
						  "expectedDetails": {},
						  "actualDetails": {
							"hwTableName": "tah_hom_luc_ucportchannelmembertable 1540",
							"hwTableField": "dst_chip, dst_port"
						  }
						}
					  }
					}
				  }
				}
			  ],
			  "skippedEntities": []
			}
		  ]
		}
	  }`

	body := make(map[string]interface{})

	//err = json.Unmarshal([]byte(insAPIResponse.InsAPI.Outputs.Output.Body), &body)

	err := json.Unmarshal([]byte(raw), &body)

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
	var payloadString string = "{\n  \"ins_api\": {\n    \"version\": \"1.0\",\n    \"type\": \"cli_show_ascii\",\n    \"chunk\": \"0\",\n    \"sid\": \"sid\",\n    \"input\": \"show consistency-checker membership vlan 2006 detail\",\n    \"output_format\": \"json\"\n  }\n}"
	var username = "admin"
	var password = "cisco!123"

	path := make([][]Node, 7)

	node0 := make([]Node, 1)
	node0_0 := Node{NodeName: "result", ToDive: false, ToCombine: false}
	node0[0] = node0_0

	node1 := make([]Node, 1)
	node1_0 := Node{NodeName: "checkers", ToDive: false, ToCombine: false}
	node1[0] = node1_0

	node2 := make([]Node, 2)
	node2_0 := Node{NodeName: "platformDetails", ToDive: true, ToCombine: false}
	node2_1 := Node{NodeName: "passedEntities", ToDive: false, ToCombine: false}
	node2[0] = node2_0
	node2[1] = node2_1

	node3 := make([]Node, 2)
	node3_0 := Node{NodeName: "key", ToDive: true, ToCombine: false}
	node3_1 := Node{NodeName: "value", ToDive: false, ToCombine: false}
	node3[0] = node3_0
	node3[1] = node3_1

	node4 := make([]Node, 2)
	node4_0 := Node{NodeName: "entityDetails", ToDive: true, ToCombine: false}
	node4_1 := Node{NodeName: "checkedProperties", ToDive: false, ToCombine: false}
	node4[0] = node4_0
	node4[1] = node4_1

	node5 := make([]Node, 1)
	node5_0 := Node{NodeName: "tahoe", ToDive: false, ToCombine: false}
	node5[0] = node5_0

	node6 := make([]Node, 2)
	node6_0 := Node{NodeName: "egrNumPaths_Sw_Hw", ToDive: false, ToCombine: true}
	node6_1 := Node{NodeName: "egrMembers_Sw_Hw", ToDive: false, ToCombine: false}
	node6[0] = node6_0
	node6[1] = node6_1

	path[0] = node0
	path[1] = node1
	path[2] = node2
	path[3] = node3
	path[4] = node4
	path[5] = node5
	path[6] = node6

	worker(url, payloadString, username, password, path)
}
