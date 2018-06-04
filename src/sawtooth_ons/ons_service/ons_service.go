package ons_service

import (
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"github.com/golang/protobuf/proto"
	"protobuf/ons_pb2"
	"sawtooth_sdk/processor"
	//"sawtooth_sdk/protobuf/processor_pb2"
	"sawtooth_sdk/logging"
	"strings"
)

var logger *logging.Logger = logging.Get()
var familyname string = "service-type"
var namespace = hexdigest(familyname)[:6]

func UnpackServiceType(service_type_byte_data []byte) (*ons_pb2.ServiceType, error) {
	service_type_data := &ons_pb2.ServiceType{}
	err := proto.Unmarshal(service_type_byte_data, service_type_data)
	logger.Debugf("unpackGS1Code service type, address : " + service_type_data.Address)

	if err != nil {
		return nil, &processor.InternalError{
			Msg: fmt.Sprint("Failed to unmarshal service type: %v", err)}
	}
	return service_type_data, nil
}

func LoadServiceType(address string, context *processor.Context) (*ons_pb2.ServiceType, error) {
	logger.Debugf("LoadServiceType address: " + address)

	//address로 state를 읽어 들인다 -> saveGS1Code에서 저장된 data이다.
	results, err := context.GetState([]string{address})

	if err != nil {
		return nil, err
	}

	if len(string(results[address])) > 0 {
		service_type_data, err := UnpackServiceType(results[address])
		if err != nil {
			return nil, err
		}

		return service_type_data, nil
	}
	return nil, nil
}

func SaveServiceType(address string, service_type_data *ons_pb2.ServiceType, context *processor.Context) error {
	data, err := proto.Marshal(service_type_data)
	if err != nil {
		return &processor.InternalError{Msg: fmt.Sprint("Failed to serialize GS1 Code data:", err)}
	}

	addresses, err := context.SetState(map[string][]byte{
		address: data,
	})
	if err != nil {
		return err
	}

	if len(addresses) == 0 {
		return &processor.InternalError{Msg: "No address in set response"}
	}

	logger.Debugf("SaveServiceType address: " + address)

	return nil
}

func DeleteServiceType(address string, context *processor.Context) error {
	//address로 state를 읽어 들인다 -> saveGS1Code에서 저장된 data이다.
	results, err := context.DeleteState([]string{address})

	if err != nil {
		return &processor.InternalError{Msg: fmt.Sprint("Failed to detele service type:", err)}
	}

	//return된 map에서 key = address에 해당하는 value가 없어야 한다.
	if len(results) == 0 {
		return &processor.InternalError{Msg: fmt.Sprint("The Service type was not deleted:", err)}
	}

	//return no error -> error is nil...
	logger.Debugf("DeleteServiceType  address : " + address)
	return nil
}

func hexdigest(str string) string {
	hash := sha512.New()
	hash.Write([]byte(str))
	hashBytes := hash.Sum(nil)
	return strings.ToLower(hex.EncodeToString(hashBytes))
}

func hexdigestbyByte(data []byte) string {
	hash := sha512.New()
	hash.Write(data)
	hashBytes := hash.Sum(nil)
	return strings.ToLower(hex.EncodeToString(hashBytes))
}

func MakeAddress(requestor string, service_type *ons_pb2.ServiceType) (string, error) {
	marshaled_service_type, err := proto.Marshal(service_type)
	if err != nil {
		return "", &processor.InternalError{Msg: fmt.Sprint("Failde to marshal service type data in MakeAddress :", err)}
	}
	return namespace + hexdigest(requestor)[:4] + hexdigestbyByte(marshaled_service_type)[:60], nil
}
