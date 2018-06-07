package main

import (
	"os"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	//"encoding/xml"
	"strings"
	"net/http"
	"bytes"
	"os/user"
	"fmt"
	"time"
	"strconv"
	"log"
	"io/ioutil"
	"bufio"
	"github.com/golang/protobuf/proto"
	xtoj "github.com/basgys/goxml2json"
	flags "github.com/jessevdk/go-flags"
	"protobuf/ons_pb2"
	"sawtooth_sdk/protobuf/transaction_pb2"
	"sawtooth_sdk/protobuf/batch_pb2"
	"sawtooth_sdk/signing"
	"test/ons_query"
)

var namespace = hexdigestbyString("ons")[:6]

var opts struct {
	Test []bool `long:"test" description:"Just for development"`
	Verbose []bool `short:"v" long:"verbose" description:"Enable verbosity"`
	GS1Code string `short:"g" long:"gs1code" description:"GS1 code for testing" default:"00800000000000"`
	Connect string `short:"c" long:"connect" description:"The validator component endpoint to" default:"http://198.13.60.39:8080"`
	RandomPrivKey []bool `short:"p" long:"random" description:"Use random private key(default key: $HOME/.sawtooth/key/$USER.priv"`
	Service string  `short:"s" long:"service" description:"Service field of NAPTR" default:"http://localhost/service.xml"`
	Regexp string  `short:"e" long:"regexp" description:"Regexp field of NAPTR" default:"!^.*$!http://example.com/cgibin/epcis!"`
	Flags rune `short:"f" long:"flags" description:"Flags field of NAPTR (default : u)" default:"117"`
	RecordIdx uint32 `short:"r" long:"recordidx" description:"The index of GS1 code's records" default:"0"`
	ServiceTypePath string `short:"x" long:"xml" description:"The service type xml or json file path" default:"./servicetype.xml"`
	ServieTypeAddress string `short:"a" long:"svcaddr" description:"The address of service type"`
	State int32 `short:"t" long:"state" description:"The state of GS1 code or record" default:"1"`
	ManagerAddress string `short:"m" long:"manager" description:"The public key to be gs1 code manager or su manager"`
}

const action_register = "register"
const action_deregister = "deregister"
const action_add = "add"
const action_remove = "remove"
const action_register_svc = "register_svc"
const action_deregister_svc = "deregister_svc"
const action_get = "get"
const action_get_svc = "get_svc"
const action_get_mngr = "get_mngr"
const action_change_gstate = "change_gstate"
const action_change_rstate = "change_rstate"
const action_add_mngr = "add_manager"

const (
	REGISTER_GS1CODE = iota+1
	DEREGISTER_GS1CODE
	ADD_RECORD
	REMOVE_RECORD
	REGISTER_SVC
	DEREGISTER_SVC
	CHANGE_GSTATE
	CHANGE_RSTATE
	ADD_MANAGER
	_
	_
	GET_GS1CODE_DATA
	GET_SVC_DATA
	GET_MNGR
)

func IfThenElse(condition bool, a interface{}, b interface{}) interface{} {
    if condition {
        return a
    }
    return b
}

func main() {
	parser := flags.NewParser(&opts, flags.Default)

	args, err := parser.Parse()
	if err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			fmt.Printf("Failed to parse args: %v\n", err)
			os.Exit(2)
		}
	}

	var is_testing bool

	switch len(opts.Test) {
	case 0:
		is_testing = false
	default:
		is_testing = true
	}

	var is_verbose bool
	switch len(opts.Verbose) {
	case 0:
		is_verbose = false
	default:
		is_verbose = true
	}

	if is_testing == true {
		is_verbose = true
	}

	var is_use_random_priv_key bool
	switch len(opts.RandomPrivKey) {
	case 0:
		is_use_random_priv_key = false
	default:
		is_use_random_priv_key = true
	}

	if len(args) == 0 {
		fmt.Println("action is needed. (atctions = register, deregister, add, remove)")
		os.Exit(2)
	}

	input_gs1_code := opts.GS1Code
	var local_private_key []byte
	var local_public_key []byte
	user, err := user.Current()
	if is_use_random_priv_key == false {
		local_private_key, err = ioutil.ReadFile(user.HomeDir+"/.sawtooth/keys/"+user.Username+".priv")
		if err != nil {
			fmt.Println("Fail to read private key.")
			os.Exit(2)
		}
		if local_private_key[len(local_private_key)-1] == 10 {
			local_private_key = local_private_key[:len(local_private_key)-1]
		}

		local_private_key, _ = hex.DecodeString(string(local_private_key))

		if is_verbose == true {
			fmt.Printf("local private key : %v\n", local_private_key)
		}

		local_public_key = nil

		}else{
		local_private_key = nil
		local_public_key = nil
	}

	var transaction_type int32
	if strings.Compare(args[0], action_register) == 0 {
		transaction_type = REGISTER_GS1CODE
	}else if strings.Compare(args[0], action_deregister) == 0 {
		transaction_type = DEREGISTER_GS1CODE
	}else if strings.Compare(args[0], action_add) == 0 {
		transaction_type = ADD_RECORD
	}else if strings.Compare(args[0], action_remove) == 0 {
		transaction_type = REMOVE_RECORD
	}else if strings.Compare(args[0], action_get) == 0 {
		transaction_type = GET_GS1CODE_DATA
	}else if strings.Compare(args[0], action_register_svc) == 0 {
		transaction_type = REGISTER_SVC
	}else if strings.Compare(args[0], action_deregister_svc) == 0 {
		transaction_type = DEREGISTER_SVC
	}else if strings.Compare(args[0], action_get_svc) == 0 {
		transaction_type = GET_SVC_DATA
	}else if strings.Compare(args[0], action_change_gstate) == 0 {
		transaction_type = CHANGE_GSTATE
	}else if strings.Compare(args[0], action_change_rstate) == 0 {
		transaction_type = CHANGE_RSTATE
	}else if args[0] == action_add_mngr {
		transaction_type = ADD_MANAGER
	}else if args[0] == action_get_mngr {
		transaction_type = GET_MNGR
	}

	if len(opts.ManagerAddress) == 0 {
		if transaction_type == ADD_MANAGER {
			fmt.Println("Need to input manager address.")
			os.Exit(2)
		}
	}

	if is_testing == true || is_verbose == true {
		fmt.Printf("command line arguments: %v\n", os.Args)
		fmt.Printf("GS1 code = %v\n", input_gs1_code)
		fmt.Printf("Service = %v\n", opts.Service)
		fmt.Printf("Regexp = %v\n", opts.Regexp)
		fmt.Printf("Flags = %c\n", opts.Flags)
		fmt.Printf("RecordIdx = %v\n", opts.RecordIdx)
		fmt.Printf("State : %d\n", opts.State)
		fmt.Printf("Test = %v\n", opts.Test)
		fmt.Printf("endpoint = %v\n", opts.Connect)
		fmt.Printf("remaining args = %v\n", args)
		fmt.Println("Username : " + user.Username)
		fmt.Println("Home Dir : " + user.HomeDir)
		fmt.Printf("transaction type = %v\n", transaction_type)
	}

	signer := MakeSigner(local_private_key, local_public_key, is_use_random_priv_key, is_testing || is_verbose)

	var address string
	var payload *ons_pb2.SendONSTransactionPayload
	var tr_err error
	switch transaction_type {
	case REGISTER_GS1CODE:
		payload, tr_err = MakeRegisterGS1CodePayload(input_gs1_code, signer.GetPublicKey().AsHex())
		address = MakeAddressByGS1Code(input_gs1_code)
	case DEREGISTER_GS1CODE:
		payload, tr_err = MakeDeregisterGS1CodePayload(input_gs1_code)
		address = MakeAddressByGS1Code(input_gs1_code)
	case ADD_RECORD:
		payload, tr_err = MakeAddRecordPayload(input_gs1_code, opts.Flags, opts.Service, opts.Regexp)
		address = MakeAddressByGS1Code(input_gs1_code)
	case REMOVE_RECORD:
		payload, tr_err = MakeRemoveRecordPayload(input_gs1_code, opts.RecordIdx)
		address = MakeAddressByGS1Code(input_gs1_code)
	case GET_GS1CODE_DATA:
		address = MakeAddressByGS1Code(input_gs1_code)
		ons_query.QueryGS1CodeData(address, opts.Connect, is_verbose)
		return
	case GET_SVC_DATA:
		ons_query.QueryServicTypeData(opts.ServieTypeAddress, opts.Connect, is_verbose)
		return
	case REGISTER_SVC:
		payload, address, tr_err = MakeRegisterServiceTypePayload(opts.ServiceTypePath, signer.GetPublicKey().AsHex(), is_verbose)
	case DEREGISTER_SVC:
		payload, tr_err = MakeDeregisterServiceTypePayload(opts.ServieTypeAddress, is_verbose)
		address = opts.ServieTypeAddress
	case CHANGE_GSTATE:
		payload, tr_err = MakeChangeGStatePayload(input_gs1_code, opts.State)
		address = MakeAddressByGS1Code(input_gs1_code)
	case CHANGE_RSTATE:
		payload, tr_err = MakeChangeRStatePayload(input_gs1_code, opts.RecordIdx, opts.State)
		address = MakeAddressByGS1Code(input_gs1_code)
	case ADD_MANAGER:
		payload, tr_err = MakeAddManagerPayload(input_gs1_code, opts.ManagerAddress)
		address = GetONSManagerAddress()
	case GET_MNGR:
		ons_query.QueryONSManager(GetONSManagerAddress(), opts.Connect, is_verbose)
		return
	default:
		payload, tr_err = MakeRegisterGS1CodePayload(input_gs1_code, signer.GetPublicKey().AsHex())
		address = MakeAddressByGS1Code(input_gs1_code)
	}

	batch_list_bytes, tr_err := MakeBatchList(payload, signer, address, is_verbose)
	if tr_err != nil {
		log.Fatal("Failed to marshal Batch list:", err)
		os.Exit(0)
	}

	if is_testing == true {
		fmt.Println("Exit program because of test option")
		os.Exit(0)
	}

	resp, err:= http.Post(opts.Connect+"/batches", "application/octet-stream", bytes.NewBuffer(batch_list_bytes))
	if err != nil {
		log.Fatal("Fail to send batch list", err)
		return;
	}

	if is_verbose == true {
		fmt.Println(resp)
	}

	defer resp.Body.Close()
	if is_verbose == true {
		fmt.Println(resp.Body)
	}
}

func MakeSigner(priv_key_str []byte, public_key_str []byte, random_priv_key bool, verify bool) (*signing.Signer) {
	context := signing.CreateContext("secp256k1")
	var private_key signing.PrivateKey
	if random_priv_key == true {
		private_key = context.NewRandomPrivateKey()
	}else{
		private_key = signing.NewSecp256k1PrivateKey(priv_key_str)
	}

	crypto_factory := signing.NewCryptoFactory(context)
	signer := crypto_factory.NewSigner(private_key)

	var public_key signing.PublicKey
	if public_key_str == nil {
		public_key = signer.GetPublicKey()
	}else{
		public_key = signing.NewSecp256k1PublicKey(public_key_str)
	}

	if verify == true {
		if random_priv_key == true {
			fmt.Printf("random private key = %v\n", private_key.AsBytes())
		}else{
			fmt.Printf("local private key  = %v (%s)\n", priv_key_str, string(priv_key_str))
		}

		fmt.Printf("signer public key  = %v %v\n", public_key.AsBytes(), public_key.AsHex())

		message := "sawtooth ons testing program"
		signature := context.Sign([]byte(message), private_key)

		if context.Verify(signature, []byte(message), public_key) == true {
			fmt.Println("Verify key pair : OK")
		}else{
			fmt.Println("Verify key pair : NOT OK")
		}
	}

	return signer
}

func MakeBatchList(transaction_payload *ons_pb2.SendONSTransactionPayload, signer *signing.Signer, address string, verbose bool) ([]byte, error) {
	gs1code_reg_payload, err := proto.Marshal(transaction_payload)
	timestamp := strconv.FormatInt(time.Now().UnixNano(), 16)
	if err != nil {
		log.Fatal("Failed to marshal transaction payload:", err)
		return nil, err
	}

	if verbose == true {
		fmt.Printf("transaction payload : %v\n", transaction_payload)
		fmt.Printf("transaction payload(marshaled) : %v\n", gs1code_reg_payload)
		fmt.Println("address : ", address)
		fmt.Println("timestamp : ", timestamp, " (it will be used as nonce)")
	}

	transaction_header := &transaction_pb2.TransactionHeader {
		FamilyName: "ons",
		FamilyVersion: "1.0",
		Inputs:  []string{address},
		Outputs: []string{address},
		BatcherPublicKey: signer.GetPublicKey().AsHex(),
		SignerPublicKey: signer.GetPublicKey().AsHex(),
		Dependencies: []string{},
		PayloadSha512: hexdigestbyByte(gs1code_reg_payload),
		Nonce: timestamp,
	}

	transaction_header_bytes, err := proto.Marshal(transaction_header)

	if err != nil {
		log.Fatal("Failed to marshal Transaction Header:", err)
		return nil, err
	}

	transaction := &transaction_pb2.Transaction {
		Header: transaction_header_bytes,
		HeaderSignature: strings.ToLower(hex.EncodeToString(signer.Sign(transaction_header_bytes))),
		Payload: gs1code_reg_payload,
	}
	if verbose == true {
		fmt.Printf("transaction : %v\n", transaction)
	}

	batch_header := &batch_pb2.BatchHeader {
		SignerPublicKey: signer.GetPublicKey().AsHex(),
		TransactionIds: []string{transaction.HeaderSignature},
	}

	batch_header_bytes, err := proto.Marshal(batch_header)
	if err != nil {
		log.Fatal("Failed to marshal Batch Header:", err)
		return nil, err
	}

	batch := &batch_pb2.Batch {
		Header: batch_header_bytes,
		HeaderSignature: strings.ToLower(hex.EncodeToString(signer.Sign(batch_header_bytes))),
		Transactions: []*transaction_pb2.Transaction{transaction},
	}

	batch_list := &batch_pb2.BatchList {
		Batches: []*batch_pb2.Batch{batch},
	}
	if verbose == true {
		fmt.Printf("batch list : %v\n", batch_list)
	}

	return proto.Marshal(batch_list)
}

func MakeRegisterGS1CodePayload(gs1_code string, owner_address string) (*ons_pb2.SendONSTransactionPayload, error){
	register_gs1_code_payload := &ons_pb2.SendONSTransactionPayload {
		TransactionType: ons_pb2.SendONSTransactionPayload_REGISTER_GS1CODE,
		RegisterGs1Code: &ons_pb2.SendONSTransactionPayload_RegisterGS1CodeTransactionData {
			Gs1Code: gs1_code,
			OwnerId: owner_address,
		},
	}
	return register_gs1_code_payload, nil
}

func MakeDeregisterGS1CodePayload(gs1_code string) (*ons_pb2.SendONSTransactionPayload, error){
	deregister_gs1_code_payload := &ons_pb2.SendONSTransactionPayload {
		TransactionType: ons_pb2.SendONSTransactionPayload_DEREGISTER_GS1CODE,
		DeregisterGs1Code: &ons_pb2.SendONSTransactionPayload_DeregisterGS1CodeTransactionData {
			Gs1Code : gs1_code,
		},
	}
	return deregister_gs1_code_payload, nil
}

func MakeAddRecordPayload(gs1_code string, flags int32, service string, regexp string) (*ons_pb2.SendONSTransactionPayload, error) {
	add_record_payload := &ons_pb2.SendONSTransactionPayload {
		TransactionType: ons_pb2.SendONSTransactionPayload_ADD_RECORD,
		AddRecord: &ons_pb2.SendONSTransactionPayload_AddRecordTransactionData {
			Gs1Code: gs1_code,
			Record: &ons_pb2.SendONSTransactionPayload_RecordTranactionData {
				Flags: flags,
				Service: service,
				Regexp: regexp,
			},
		},
	}
	return add_record_payload, nil
}

func MakeRemoveRecordPayload(gs1_code string, remove_idx uint32) (*ons_pb2.SendONSTransactionPayload, error) {
	remove_record_payload := &ons_pb2.SendONSTransactionPayload {
		TransactionType: ons_pb2.SendONSTransactionPayload_REMOVE_RECORD,
		RemoveRecord: &ons_pb2.SendONSTransactionPayload_RemoveRecordTransactionData {
			Gs1Code: gs1_code,
			Index: remove_idx,
		},
	}
	return remove_record_payload, nil
}

func MakeRegisterServiceTypePayload(file_path string, requestor string, verbose bool) (*ons_pb2.SendONSTransactionPayload, string, error) {
	service_type, address, err := GenerateServiceType(file_path, requestor, verbose)
	if err != nil {
		log.Fatal("Failed to GenerateServiceType :", err)
		return nil, "", err
	}

	register_service_type_payload := &ons_pb2.SendONSTransactionPayload {
		TransactionType: ons_pb2.SendONSTransactionPayload_REGISTER_SERVICETYPE,
		RegisterServiceType: &ons_pb2.SendONSTransactionPayload_RegisterServiceTypeTransactionData {
			Address: address,
			ServiceType: service_type,
		},
	}
	return register_service_type_payload, address, nil
}

func MakeDeregisterServiceTypePayload(address string, verbose bool) (*ons_pb2.SendONSTransactionPayload, error) {
	deregister_service_type_payload := &ons_pb2.SendONSTransactionPayload {
		TransactionType: ons_pb2.SendONSTransactionPayload_DEREGISTER_SERVICETYPE,
		DeregisterServiceType: &ons_pb2.SendONSTransactionPayload_DeregisterServiceTypeTransactionData {
			Address: address,
		},
	}
	return deregister_service_type_payload, nil
}

func MakeChangeGStatePayload(gs1_code string, state int32) (*ons_pb2.SendONSTransactionPayload, error){
	change_gs1_code_state_payload := &ons_pb2.SendONSTransactionPayload {
		TransactionType: ons_pb2.SendONSTransactionPayload_CHANGE_GS1CODE_STATE,
		ChangeGs1CodeState: &ons_pb2.SendONSTransactionPayload_ChangeGS1CodeStateTransactionData {
			Gs1Code : gs1_code,
			State: ons_pb2.GS1CodeData_GS1CodeState(state),
		},
	}
	return change_gs1_code_state_payload, nil
}

func MakeChangeRStatePayload(gs1_code string, record_idx uint32, state int32) (*ons_pb2.SendONSTransactionPayload, error){
	change_record_state_payload := &ons_pb2.SendONSTransactionPayload {
		TransactionType: ons_pb2.SendONSTransactionPayload_CHANGE_RECORD_STATE,
		ChangeRecordState: &ons_pb2.SendONSTransactionPayload_ChangeRecordStateTransactionData {
			Gs1Code : gs1_code,
			Index: record_idx,
			State: ons_pb2.Record_RecordState(state),
		},
	}
	return change_record_state_payload, nil
}

func MakeAddManagerPayload(gs1_code string, address string) (*ons_pb2.SendONSTransactionPayload, error){
	tmp := &ons_pb2.SendONSTransactionPayload {
		TransactionType: ons_pb2.SendONSTransactionPayload_ADD_MANAGER,
		AddManager: &ons_pb2.SendONSTransactionPayload_AddManagerTransactionData {
			Gs1Code : gs1_code,
			Address: address,
		},
	}
	return tmp, nil
}

func hexdigestbyString(str string) string {
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

func MakeAddressByGS1Code(gs1_code string) string{
	return namespace + hexdigestbyString(gs1_code)[:64]
}

func GetONSManagerAddress() string {
	return namespace + hexdigestbyString("ons_manager")[:64]
}

func MakeAddressByServiceType(requestor string, service_type *ons_pb2.ServiceType) (string, error) {
	marshaled_service_type, err := proto.Marshal(service_type)
	if err != nil {
		return "", err
	}
	return namespace + hexdigestbyString("service-type")[:8] + hexdigestbyString(requestor)[:16] + hexdigestbyByte(marshaled_service_type)[:40], nil
}

func CheckXmlFileType(out *os.File) bool {
    // Only the first 512 bytes are used to sniff the content type.
    buffer := make([]byte, 512)

    _, err := out.Read(buffer)
    if err != nil {
        return false
    }

    // Use the net/http package's handy DectectContentType function. Always returns a valid
    // content-type by returning "application/octet-stream" if no others seemed to match.
	contentType := http.DetectContentType(buffer)

	if strings.Index(strings.ToLower(contentType), "xml") != -1 {
		fmt.Println(contentType)
		return true
	}

    return false
}

func GenerateServiceType(file_path string, requestor string, verbose bool) (*ons_pb2.ServiceType, string, error) {
	var fields map[string]interface{}
	var json_raw_data []byte
	//tr_fields, tr_type은 transaction으로 전달하는 json을 만들기 위한 변수임.
	tr_fields := []*ons_pb2.ServiceType_ServiceTypeField{}
	tr_types  := []*ons_pb2.ServiceType_ServiceTypeField{}

	f, err := os.Open(file_path)
    if err != nil {
		fmt.Println("os.Open error : ", err)
		return nil, "", err
    }
    defer f.Close()

	if CheckXmlFileType(f) == true {
		_, err := f.Seek(0, 0)
		if err != nil {
			fmt.Println("file seek error : ", err)
			return nil, "", err
		}

		r := bufio.NewReader(f)
		js, err := xtoj.Convert(r)
		if err != nil {
			fmt.Println("xtoj.Convert error : ", err)
			return nil, "", err
		}
		json_raw_data = js.Bytes()
	}else{
		if verbose == true {
			fmt.Printf("%s type isn't xml, so directly make json object from the file\n", file_path)
		}
		var err error
		json_raw_data, err = ioutil.ReadFile(file_path)
		if err != nil {
			fmt.Printf("ioutil.ReadFile error : %v\n", err)
			return nil, "", err
		}
	}

	if verbose == true {
		fmt.Println(string(json_raw_data))
	}

	err = json.Unmarshal(json_raw_data, &fields)
	if err != nil {
		fmt.Printf("json.Unmarshal error : %v\n", err)
		return nil, "", err
	}

	for k, v := range fields {
		switch vv := v.(type) {
		case map[string]interface{}:
			if k == "ServiceType" {
				for _k, _v := range vv {
					switch _vv := _v.(type) {
					case string:
						new_field := &ons_pb2.ServiceType_ServiceTypeField{
							Key: strings.Trim(_k, "-"),
							Value: _vv,
						}
						tr_fields = append(tr_fields, new_field)
					default:
						_tmp_json, err := json.Marshal(_vv)
						if err == nil {
							tmp_key := strings.Trim(_k, "-")
							new_field := &ons_pb2.ServiceType_ServiceTypeField{
								Key: tmp_key,
								Value: string(_tmp_json),
							}
							new_type := &ons_pb2.ServiceType_ServiceTypeField{
								Key: tmp_key,
								Value: "json",
							}
							tr_fields = append(tr_fields, new_field)
							tr_types = append(tr_types, new_type)
						}else{
							fmt.Printf("Failed to json marshal from %s key's value(%s), so it will be skipped: error : %v\n",
									_k, _vv, err)
						}
					}
				}
			}
		}
	}

	if verbose == true {
		fmt.Println("tr_fields \n", tr_fields)
		fmt.Println("tr_type \n", tr_types)
	}

	svc_type := &ons_pb2.ServiceType {
		Fields: tr_fields,
		Types: tr_types,
	}

	address, err := MakeAddressByServiceType(requestor, svc_type)
	if err != nil {
		fmt.Printf("error : %v\n", err)
		return nil, "", err
	}

	svc_type.Address = address
	svc_type.Provider = requestor

	return svc_type, address, nil
}