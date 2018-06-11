package ons_query

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"encoding/json"
	"encoding/base64"
	"protobuf/ons_pb2"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/jsonpb"
)

func GetRawData(address string, url string, verbose bool) ([]byte, error) {
	get_url := url + "/state/" + address

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

	return pb2_data, nil
}

func PrintPrettyJson(pb proto.Message, verbose bool) error {
	m := &jsonpb.Marshaler{}
	json_string, err := m.MarshalToString(pb)
	if err != nil {
		fmt.Printf("Fail to convert proto.Message to json string: %v\n", err)
		return err
	}

	if verbose == true {
		fmt.Printf("json : %v\n", json_string)
	}

	var dat map[string] interface{}
    if err := json.Unmarshal([]byte(json_string), &dat); err != nil {
		fmt.Printf("PrintPrettyJson : json.Unmarshal : error %v\n", err);
		return err
	}

	b, err := json.MarshalIndent(dat, "", "  ")
    if err != nil {
		fmt.Printf("PrintPrettyJson : json.MarshalIndent : error %v\n", err);
		return err
	}

    b2 := append(b, '\n')
	fmt.Printf(string(b2))
	return nil
}

func QueryGS1CodeData(gs1_code_address string, url string, verbose bool) (*ons_pb2.GS1CodeData, error) {
	pb2_data, err := GetRawData(gs1_code_address, url, verbose)
	if err != nil {
		return nil, err
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

	_ = PrintPrettyJson(gs1_code_data, verbose)

	return gs1_code_data, nil
}

func QueryServicTypeData(service_type_address string, url string, verbose bool) (*ons_pb2.ServiceType, error) {
	pb2_data, err := GetRawData(service_type_address, url, verbose)
	if err != nil {
		return nil, err
	}

	svc_type_data := &ons_pb2.ServiceType{}
	err = proto.Unmarshal(pb2_data, svc_type_data)
	if err != nil {
		fmt.Printf("Fail to unmarshal service type data : %v\n", err)
		return nil, err
	}

	if verbose == true {
		fmt.Printf("protobuf unmarshaled data : %v\n", svc_type_data)
	}

	_ = PrintPrettyJson(svc_type_data, verbose)

	return svc_type_data, nil
}


func QueryONSManager(ons_manager_address string, url string, verbose bool) (*ons_pb2.ONSManager, error) {
	pb2_data, err := GetRawData(ons_manager_address, url, verbose)
	if err != nil {
		return nil, err
	}

	ons_manager:= &ons_pb2.ONSManager{}
	err = proto.Unmarshal(pb2_data, ons_manager)
	if err != nil {
		fmt.Printf("Fail to unmarshal GS1 code data : %v\n", err)
		return nil, err
	}

	if verbose == true {
		fmt.Printf("protobuf unmarshaled data : %v\n", ons_manager_address)
	}

	_ = PrintPrettyJson(ons_manager, verbose)

	return ons_manager, nil
}