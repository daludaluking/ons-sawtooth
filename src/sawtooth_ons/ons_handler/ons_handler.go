package ons_handler

import (
	"fmt"
	"strings"
	"protobuf/ons_pb2"
	"sawtooth_ons/ons_state"
	"sawtooth_ons/ons_service"
	"sawtooth_ons/ons_manager"
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

func (self *ONSHandler) SetSudoAddress(address string) bool {
	return ons_manager.SetSudoAddress(address)
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
	case ons_pb2.SendONSTransactionPayload_CHANGE_GS1CODE_STATE:
		return applyChangeGS1CodeState(payload.ChangeGs1CodeState, context, requestor_pk)
	case ons_pb2.SendONSTransactionPayload_CHANGE_RECORD_STATE:
		return applyChangeRecordState(payload.ChangeRecordState, context, requestor_pk)
	case ons_pb2.SendONSTransactionPayload_ADD_MANAGER:
		return applyAddManager(payload.AddManager, context, requestor_pk)
	case ons_pb2.SendONSTransactionPayload_REMOVE_MANAGER:
		return applyRemoveManager(payload.RemoveManager, context, requestor_pk)
	case ons_pb2.SendONSTransactionPayload_ADD_SUMANAGER:
		return applyAddSuManager(payload.AddSumanager, context, requestor_pk)
	case ons_pb2.SendONSTransactionPayload_REMOVE_SUMANAGER:
		return applyRemoveSuManager(payload.RemoveSumanager, context, requestor_pk)
	default:
		return &processor.InvalidTransactionError{
			Msg: fmt.Sprintf("Invalid TransactionType: '%v'", payload.TransactionType)}
	}
}

func applyRegiserGS1Code(
	registerGS1CodeData *ons_pb2.SendONSTransactionPayload_RegisterGS1CodeTransactionData,
	context *processor.Context,	requestor string) error {
	//permission check...
	if GetPermissionLevel("", requestor, ons_manager.PERMISSION_SU_MANAGER, context) == false {
		return &processor.InvalidTransactionError{Msg: "applyRegiserGS1Code : Authentication failed"}
	}

	gs1_code_data, err := ons_state.LoadGS1Code(registerGS1CodeData.GetGs1Code(), context)
	if err != nil {
		return err
	}

	if gs1_code_data != nil {
		return &processor.InvalidTransactionError{Msg: "GS1 Code already exists"}
	}

	new_gs1_code := &ons_pb2.GS1CodeData{
		Gs1Code: registerGS1CodeData.GetGs1Code(),
		OwnerId: registerGS1CodeData.GetOwnerId(),
		State: ons_pb2.GS1CodeData_GS1CODE_INACTIVE,
	}

	return ons_state.SaveGS1Code(new_gs1_code, context)
}

func applyDeregiserGS1Code(
	deregisterGS1CodeData *ons_pb2.SendONSTransactionPayload_DeregisterGS1CodeTransactionData,
	context *processor.Context,
	requestor string) error {
	//permission check...
	if GetPermissionLevel("", requestor, ons_manager.PERMISSION_SU_MANAGER, context) == false {
		return &processor.InvalidTransactionError{Msg: "applyDeregiserGS1Code : Authentication failed"}
	}

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
	//permission check...
	if GetPermissionLevel(addRecordData.GetGs1Code(), requestor, ons_manager.PERMISSION_MANAGER, context) == false {
		return &processor.InvalidTransactionError{Msg: "applyAddRecord : Authentication failed"}
	}

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
	//permission check...
	if GetPermissionLevel(removeRecordData.GetGs1Code(), requestor, ons_manager.PERMISSION_MANAGER, context) == false {
		return &processor.InvalidTransactionError{Msg: "applyRemoveRecord : Authentication failed"}
	}

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
	if gs1_code_data.Records[idx].Provider != requestor {
		return &processor.InvalidTransactionError{Msg: "applyRemoveRecord : mismatch provider address"}
	}

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
	//permission check...
	if GetPermissionLevel("", requestor, ons_manager.PERMISSION_SU_MANAGER, context) == false {
		return &processor.InvalidTransactionError{Msg: "applyRegiserServiceType : Authentication failed"}
	}

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
	//permission check...
	if GetPermissionLevel("", requestor, ons_manager.PERMISSION_SU_MANAGER, context) == false {
		return &processor.InvalidTransactionError{Msg: "applyDeregiserServiceType : Authentication failed"}
	}

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

func applyChangeGS1CodeState(
	changeGS1CodeState *ons_pb2.SendONSTransactionPayload_ChangeGS1CodeStateTransactionData,
	context *processor.Context,
	requestor string) error {
	//permission check...
	if GetPermissionLevel("", requestor, ons_manager.PERMISSION_SU_MANAGER, context) == false {
		return &processor.InvalidTransactionError{Msg: "applyChangeGS1CodeState : Authentication failed"}
	}

	gs1_code_data, err := ons_state.LoadGS1Code(changeGS1CodeState.GetGs1Code(), context)
	if err != nil {
		return err
	}

	if gs1_code_data == nil {
		return &processor.InvalidTransactionError{Msg: "GS1 Code doesn't exist"}
	}

	gs1_code_data.State = changeGS1CodeState.GetState()

	return ons_state.SaveGS1Code(gs1_code_data, context)
}

func applyChangeRecordState(
	changeRecordState *ons_pb2.SendONSTransactionPayload_ChangeRecordStateTransactionData,
	context *processor.Context,
	requestor string) error {
	//permission check...
	if GetPermissionLevel("", requestor, ons_manager.PERMISSION_SU_MANAGER, context) == false {
		return &processor.InvalidTransactionError{Msg: "applyChangeRecordState : Authentication failed"}
	}

	gs1_code_data, err := ons_state.LoadGS1Code(changeRecordState.GetGs1Code(), context)
	if err != nil {
		return err
	}

	if gs1_code_data == nil {
		return &processor.InvalidTransactionError{Msg: "GS1 Code doesn't exist"}
	}

	idx := changeRecordState.GetIndex()
	record_len := uint32(len(gs1_code_data.Records))

	//permissino check??
	//ons_pb2.SendONSTransactionPayload_RecordTranactionData
	//ons_pb2.Record
	if record_len <= idx {
		return &processor.InvalidTransactionError{Msg: "Invalid index: " + string(idx) + ", record count: " + string(record_len)}
	}

	gs1_code_data.Records[idx].State = changeRecordState.GetState()
	return ons_state.SaveGS1Code(gs1_code_data, context)
}

func applyAddManager(
	addManagerData *ons_pb2.SendONSTransactionPayload_AddManagerTransactionData,
	context *processor.Context,
	requestor string) error {

	//permission check...
	if GetPermissionLevel("", requestor, ons_manager.PERMISSION_SU_MANAGER, context) == false {
		return &processor.InvalidTransactionError{Msg: "applyChangeRecordState : Authentication failed"}
	}

	//just for test...
	if addManagerData.GetGs1Code() == "0" {
		logger.Debugf("Delete manager global state")
		return ons_manager.DeleteAllManager(context)
	}

	//GS1Code Manager의 경우에는 권한이 SU Address거나 SU Manager, 또는 gs1 code manager 자신의 경우에는
	//등록, 삭제, 수정이 가능하다.
	return ons_manager.AddGS1CodeManager(addManagerData.GetGs1Code(), addManagerData.GetAddress(), requestor, context)
}

func applyRemoveManager(
	removeManagerData *ons_pb2.SendONSTransactionPayload_RemoveManagerTransactionData,
	context *processor.Context,
	requestor string) error {
	//permission check...
	if GetPermissionLevel("", requestor, ons_manager.PERMISSION_SU_MANAGER, context) == false {
		return &processor.InvalidTransactionError{Msg: "applyChangeRecordState : Authentication failed"}
	}

	//GS1Code Manager의 경우에는 권한이 SU Address거나 SU Manager의 경우에는
	//등록, 삭제, 수정이 가능하다.
	return ons_manager.RemoveGS1CodeManager(removeManagerData.GetGs1Code(), requestor, context)
}

func applyAddSuManager(
	addSuManagerData *ons_pb2.SendONSTransactionPayload_AddSUManagerTransactionData,
	context *processor.Context,
	requestor string) error {
	//permission check...
	if GetPermissionLevel("", requestor, ons_manager.PERMISSION_SU_ADDRESS, context) == false {
		return &processor.InvalidTransactionError{Msg: "applyChangeRecordState : Authentication failed"}
	}

	return ons_manager.AddSuManager(addSuManagerData.GetAddress(), requestor, context)
}

func applyRemoveSuManager(
	removeSuManagerData *ons_pb2.SendONSTransactionPayload_RemoveSUManagerTransactionData,
	context *processor.Context,
	requestor string) error {
	//permission check...
	if GetPermissionLevel("", requestor, ons_manager.PERMISSION_SU_ADDRESS, context) == false {
		return &processor.InvalidTransactionError{Msg: "applyChangeRecordState : Authentication failed"}
	}

	//GS1Code Manager의 경우에는 권한이 SU Address거나 SU Manager의 경우에는
	//등록, 삭제, 수정이 가능하다.
	return ons_manager.RemoveSuManager(removeSuManagerData.GetAddress(), requestor, context)
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

func GetPermissionLevel(gs1_code string, requestor string, require_perm ons_manager.Permission, context *processor.Context) bool{
	permission, err:= ons_manager.CheckPermission(gs1_code, requestor, context)

	if err != nil {
		logger.Debugf("Failed to check permission")
		return false
	}

	//permission is requestor's permission
	//요청자가 가지는 permission level이 요구하는 permission level보다 작거나 같으면
	//요청자는 원하는 permission level을 가지고 있는 것이다.
	if permission <= require_perm {
		return true
	}
	return false
}
