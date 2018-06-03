package ons_state

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
var familyname string = "ons"
var namespace = Hexdigest(familyname)[:6]

func UnpackPayload(payloadData []byte) (*ons_pb2.SendONSTransactionPayload, error) {
	payload := &ons_pb2.SendONSTransactionPayload{}
	err := proto.Unmarshal(payloadData, payload)
	if err != nil {
		return nil, &processor.InternalError{
			Msg: fmt.Sprint("Failed to unmarshal ONSTransaction: %v", err)}
	}
	return payload, nil
}

func UnpackGS1Code(gs1_code_byte_data []byte) (*ons_pb2.GS1CodeData, error) {
	gs1_code_data := &ons_pb2.GS1CodeData{}
	err := proto.Unmarshal(gs1_code_byte_data, gs1_code_data)
	logger.Debugf("unpackGS1Code gs1 code : " +  gs1_code_data.Gs1Code)

	if err != nil {
		return nil, &processor.InternalError{
			Msg: fmt.Sprint("Failed to unmarshal GS1 Code: %v", err)}
	}
	return gs1_code_data, nil
}

func LoadGS1Code(gs1_code string, context *processor.Context) (*ons_pb2.GS1CodeData, error) {
	//namespac와 gs1 code로 address를 만든다.
	address := MakeAddress(gs1_code)
	logger.Debugf("loadGS1Code gs1code: " +  gs1_code + ", address : " + address)

	//address로 state를 읽어 들인다 -> saveGS1Code에서 저장된 data이다.
	results, err := context.GetState([]string{address})

	if err != nil {
		return nil, err
	}

	if len(string(results[address])) > 0 {
		gs1_code_data, err := UnpackGS1Code(results[address])
		if err != nil {
			return nil, err
		}

		return gs1_code_data, nil
	}
	return nil, nil
}

func SaveGS1Code(gs1_code_data *ons_pb2.GS1CodeData, context *processor.Context) error {
	address := MakeAddress(gs1_code_data.GetGs1Code())
	data, err := proto.Marshal(gs1_code_data)
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
		return &processor.InternalError{Msg: "No addresses in set response"}
	}

	logger.Debugf("SaveGS1Code gs1code: " +  gs1_code_data.GetGs1Code() + ", address : " + address)

	return nil
}

func DeleteGS1Code(gs1_code string, context *processor.Context) error {
	//namespac와 gs1 code로 address를 만든다.
	address := MakeAddress(gs1_code)

	//address로 state를 읽어 들인다 -> saveGS1Code에서 저장된 data이다.
	results, err := context.DeleteState([]string{address})

	if err != nil {
		return &processor.InternalError{Msg: fmt.Sprint("Failed to detele GS1 Code data:", err)}
	}

	//return된 map에서 key = address에 해당하는 value가 없어야 한다.
	if len(results) == 0 {
		return &processor.InternalError{Msg: fmt.Sprint("GS1 Code data was not deleted:", err)}
	}

	//return no error -> error is nil...
	logger.Debugf("DeleteGS1Code gs1code: " +  gs1_code + ", address : " + address)
	return nil
}

func Hexdigest(str string) string {
	hash := sha512.New()
	hash.Write([]byte(str))
	hashBytes := hash.Sum(nil)
	return strings.ToLower(hex.EncodeToString(hashBytes))
}

func MakeAddress(gs1_code string) string{
	return namespace + Hexdigest(gs1_code)[:64]
}

func GetNameSapce() string{
	return namespace;
}

func GetFamilyName() string{
	return familyname;
}

func GetFamilyVersion() string{
	return "1.0"
}