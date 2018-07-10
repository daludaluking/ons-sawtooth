package ons_service

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/daludaluking/ons-sawtooth-sdk/ons_pb2"
	"github.com/daludaluking/ons-sawtooth-sdk/processor"
	"github.com/daludaluking/ons-sawtooth-sdk/logging"
)

var logger *logging.Logger = logging.Get()

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
		logger.Debugf("Failed to LoadServiceType(1): " + address)
		return nil, err
	}

	if len(string(results[address])) > 0 {
		service_type_data, err := UnpackServiceType(results[address])
		if err != nil {
			logger.Debugf("Failed to LoadServiceType(2): " , address)
			return nil, &processor.InvalidTransactionError{Msg: "Faied to UnpackServiceType, address: " + address}
		}

		return service_type_data, nil
	}
	return nil, nil
}

func CheckAddress(address string, context *processor.Context) bool {
	logger.Debugf("CheckAddress address: " + address)

	//address로 state를 읽어 들인다 -> saveGS1Code에서 저장된 data이다.
	results, err := context.GetState([]string{address})

	if err != nil {
		logger.Debugf("Failed to CheckAddress(1): " + address)
		return false
	}

	if len(string(results[address])) > 0 {
		return true
	}

	return false
}

func SaveServiceType(address string, service_type_data *ons_pb2.ServiceType, context *processor.Context) error {
	data, err := proto.Marshal(service_type_data)
	if err != nil {
		return &processor.InternalError{Msg: fmt.Sprint("Failed to serialize service type data:", err)}
	}

	logger.Debugf("data length : %v", len(data))

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
