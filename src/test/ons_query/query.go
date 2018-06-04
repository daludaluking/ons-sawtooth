package ons_query

import (
/*
	flags "github.com/jessevdk/go-flags"
	"os"
	"crypto/sha512"
	"encoding/hex"
	"log"
	"github.com/golang/protobuf/proto"
	"protobuf/ons_pb2"
	"sawtooth_sdk/protobuf/transaction_pb2"
	"sawtooth_sdk/protobuf/batch_pb2"
	"sawtooth_sdk/signing"
	"strings"
	"net/http"
	"encoding/json"
	"bytes"
	"os/user"
	"fmt"
	"io/ioutil"
	"time"
	"strconv"
*/
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/jsonpb"
	"encoding/json"
	"encoding/base64"
	"protobuf/ons_pb2"
	"io/ioutil"
	"net/http"
	"fmt"
)

func QueryGS1CodeData(gs1_code_address string, url string, verbose bool) (*ons_pb2.GS1CodeData, error) {
	get_url := url + "/state/" + gs1_code_address

	if verbose == true {
		fmt.Println("query : " + get_url)
	}

	resp, err := http.Get(get_url)

	if err != nil {
		fmt.Printf("Fail to query : %v\n", err)
		return nil, err
	}

	defer resp.Body.Close()

	if verbose == true {
		fmt.Printf("reponse body : %v\n", resp.Body)
	}

	data, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		fmt.Printf("Fail to read body : %v\n", err)
		return nil, err
	}

	if verbose == true {
		fmt.Printf("readed data : %s\n", string(data))
	}

	//data is json format...
	//var json_data map[string]string
	var json_data map[string]interface{}
	if err := json.Unmarshal(data, &json_data); err != nil {
		fmt.Printf("Fail to json unmarshal : %v\n", err)
		return nil, err
	}

	if json_data["error"] != nil {
		error_data := json_data["error"].(map[string]interface{})
		fmt.Printf("Fail to query: %s, %s (error code = %d)\n",
					error_data["title"].(string),
					error_data["message"].(string),
					int32(error_data["code"].(float64)))
		return nil, nil
	}

	if verbose == true {
		fmt.Printf("data : %v\n", json_data["data"])
	}

	pb2_data, err := base64.StdEncoding.DecodeString(json_data["data"].(string))

	if err != nil {
		fmt.Printf("Fail to base64 decoding : %v\n", err)
		return nil, err
	}

	if verbose == true {
		fmt.Printf("protobuf marshaled data : %q\n", pb2_data)
	}

	gs1_code_data := &ons_pb2.GS1CodeData{}
	err = proto.Unmarshal(pb2_data, gs1_code_data)
	if err != nil {
		fmt.Printf("Fail to unmarshal GS1 code data : %v\n", err)
		return nil, err
	}

	if verbose == true {
		fmt.Printf("protobuf unmarshaled data : %v\n", gs1_code_data)
	}

	m := &jsonpb.Marshaler{}
	gs1_code_data_string, err := m.MarshalToString(gs1_code_data)
	if err != nil {
		fmt.Printf("Fail to convert GS1 code data to json string: %v\n", err)
		return nil, err
	}

	if verbose == true {
		fmt.Printf("gs1 code data : %v\n", gs1_code_data_string)
	}

	pretty_formatted_data, err := json.MarshalIndent(gs1_code_data, "", "  ")
	if err == nil {
			fmt.Println("formatted : \n", string(pretty_formatted_data))
	}
	return gs1_code_data, nil
}