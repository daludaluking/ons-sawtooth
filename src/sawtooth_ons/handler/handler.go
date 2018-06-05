package handler

import (
	"fmt"
	"strings"
	"protobuf/ons_pb2"
	"sawtooth_ons/ons_state"
	"sawtooth_ons/ons_service"
	"sawtooth_sdk/logging"
	"sawtooth_sdk/processor"
	"sawtooth_sdk/protobuf/processor_pb2"
	"github.com/golang/protobuf/proto"
)

var logger *logging.Logger = logging.Get()

type ONSHandler struct {
}

func (self *ONSHandler) FamilyName() string {
	return ons_state.GetFamilyName()
}

func (self *ONSHandler) FamilyVersions() []string {
	return []string{ons_state.GetFamilyVersion()}
}

func (self *ONSHandler) Namespaces() []string {
	return []string{ons_state.GetNameSapce()}
}

func (self *ONSHandler) Apply(request *processor_pb2.TpProcessRequest, context *processor.Context) error {

	requestor_pk := request.GetHeader().GetSignerPublicKey()
	payload, err := UnpackPayload(request.GetPayload())

	logger.Debugf("call apply from ", requestor_pk)

	if err != nil {
		return err
	}

	logger.Debugf("ONS txn %v: type %v", request.Signature, payload.TransactionType)

	switch payload.TransactionType {
	case ons_pb2.SendONSTransactionPayload_REGISTER_GS1CODE:
		return applyRegiserGS1Code(payload.RegisterGs1Code, context, requestor_pk)
	case ons_pb2.SendONSTransactionPayload_DEREGISTER_GS1CODE:
		return applyDeregiserGS1Code(payload.DeregisterGs1Code, context, requestor_pk)
	case ons_pb2.SendONSTransactionPayload_ADD_RECORD:
		return applyAddRecord(payload.AddRecord, context, requestor_pk)
	case ons_pb2.SendONSTransactionPayload_REMOVE_RECORD:
		return applyRemoveRecord(payload.RemoveRecord, context, requestor_pk)
	case ons_pb2.SendONSTransactionPayload_REGISTER_SERVICETYPE:
		return applyRegiserServiceType(payload.RegisterServiceType, context, requestor_pk)
	case ons_pb2.SendONSTransactionPayload_DEREGISTER_SERVICETYPE:
		return applyDeregiserServiceType(payload.DeregisterServiceType, context, requestor_pk)
	default:
		return &processor.InvalidTransactionError{
			Msg: fmt.Sprintf("Invalid TransactionType: '%v'", payload.TransactionType)}
	}
}

func applyRegiserGS1Code(
	registerGS1CodeData *ons_pb2.SendONSTransactionPayload_RegisterGS1CodeTransactionData,
	context *processor.Context,
	requestor string) error {
	gs1_code_data, err := ons_state.LoadGS1Code(registerGS1CodeData.GetGs1Code(), context)
	if err != nil {
		return err
	}

	if gs1_code_data != nil {
		return &processor.InvalidTransactionError{Msg: "GS1 Code already exists"}
	}

	new_gs1_code := &ons_pb2.GS1CodeData{
		Gs1Code: registerGS1CodeData.GetGs1Code(),
		OwnerId: requestor,
	}

	return ons_state.SaveGS1Code(new_gs1_code, context)
}

func applyDeregiserGS1Code(
	deregisterGS1CodeData *ons_pb2.SendONSTransactionPayload_DeregisterGS1CodeTransactionData,
	context *processor.Context,
	requestor string) error {
	gs1_code_data, err := ons_state.LoadGS1Code(deregisterGS1CodeData.GetGs1Code(), context)
	if err != nil {
		return err
	}

	if gs1_code_data == nil {
		return &processor.InvalidTransactionError{Msg: "GS1 Code doesn't exist"}
	}

	if strings.Compare(gs1_code_data.GetOwnerId(), requestor) != 0 {
		return &processor.InvalidTransactionError{Msg: "Requestor's public key doesn't match with owner pubic key of GS1 Code"}
	}

	return ons_state.DeleteGS1Code(deregisterGS1CodeData.GetGs1Code(), context)
}

func applyAddRecord(
	addRecordData *ons_pb2.SendONSTransactionPayload_AddRecordTransactionData,
	context *processor.Context,
	requestor string) error {
	gs1_code_data, err := ons_state.LoadGS1Code(addRecordData.GetGs1Code(), context)
	if err != nil {
		return err
	}

	if gs1_code_data == nil {
		return &processor.InvalidTransactionError{Msg: "GS1 Code doesn't exist"}
	}

	fmt.Printf("%v\n", gs1_code_data)

	//permissino check??
	//ons_pb2.SendONSTransactionPayload_RecordTranactionData
	//ons_pb2.Record
	new_record := &ons_pb2.Record{
		Flags:    addRecordData.GetRecord().GetFlags(),
		Service:  addRecordData.GetRecord().GetService(),
		Regexp:   addRecordData.GetRecord().GetRegexp(),
		State:    ons_pb2.Record_RECORD_INACTIVE,
		Provider: requestor,
	}

	if gs1_code_data.Records == nil {
		gs1_code_data.Records = []*ons_pb2.Record{new_record}
	} else {
		gs1_code_data.Records = append(gs1_code_data.Records, new_record)
	}

	return ons_state.SaveGS1Code(gs1_code_data, context)
}

func applyRemoveRecord(
	removeRecordData *ons_pb2.SendONSTransactionPayload_RemoveRecordTransactionData,
	context *processor.Context,
	requestor string) error {
	gs1_code_data, err := ons_state.LoadGS1Code(removeRecordData.GetGs1Code(), context)
	if err != nil {
		return err
	}

	if gs1_code_data == nil {
		return &processor.InvalidTransactionError{Msg: "GS1 Code doesn't exist"}
	}

	idx := removeRecordData.GetIndex()
	record_len := uint32(len(gs1_code_data.Records))

	//permissino check??
	//ons_pb2.SendONSTransactionPayload_RecordTranactionData
	//ons_pb2.Record
	if record_len <= idx {
		return &processor.InvalidTransactionError{Msg: "Invalid index: " + string(idx) + ", record count: " + string(record_len)}
	}

	gs1_code_data.Records = append(gs1_code_data.Records[0:idx], gs1_code_data.Records[idx+1:]...)

	return ons_state.SaveGS1Code(gs1_code_data, context)
}

func applyRegiserServiceType(
	registerServiceType *ons_pb2.SendONSTransactionPayload_RegisterServiceTypeTransactionData,
	context *processor.Context,
	requestor string) error {
	//service_type := registerServiceType.ServiceType
	address := registerServiceType.Address

	tmp_data, err := ons_service.LoadServiceType(address, context)

	if err != nil {
		logger.Debugf("Failed to LoadServiceType address: " + address)
		return err;
	}

	if tmp_data != nil {
		return &processor.InvalidTransactionError{Msg: "The same service type already exists"}
	}

	return ons_service.SaveServiceType(address, registerServiceType.ServiceType, context)
}

func applyDeregiserServiceType(
	deregisterServiceType *ons_pb2.SendONSTransactionPayload_DeregisterServiceTypeTransactionData,
	context *processor.Context,
	requestor string) error {
	address := deregisterServiceType.Address
	tmp_data, err := ons_service.LoadServiceType(address, context)

	if err != nil {
		return err;
	}

	if tmp_data == nil {
		return &processor.InvalidTransactionError{Msg: "The service type doesn't exists"}
	}

	if strings.Compare(tmp_data.GetProvider(), requestor) != 0 {
		return &processor.InvalidTransactionError{Msg: "Requestor's public key doesn't match with provider pubic key of Service Type"}
	}

	return ons_service.DeleteServiceType(address, context)
}

func UnpackPayload(payloadData []byte) (*ons_pb2.SendONSTransactionPayload, error) {
	payload := &ons_pb2.SendONSTransactionPayload{}
	err := proto.Unmarshal(payloadData, payload)
	if err != nil {
		return nil, &processor.InternalError{
			Msg: fmt.Sprint("Failed to unmarshal ONSTransaction: %v", err)}
	}
	return payload, nil
}
