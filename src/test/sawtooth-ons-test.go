package main

import (
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
	"test/ons_query"
	"strings"
	"net/http"
	"bytes"
	"os/user"
	"fmt"
	"io/ioutil"
	"time"
	"strconv"
)

var namespace = hexdigestbyString("ons")[:6]

var opts struct {
	Test []bool `short:"t" long:"test" description:"Just for development"`
	Verbose []bool `short:"v" long:"verbose" description:"Enable verbosity"`
	GS1Code string `short:"g" long:"gs1code" description:"GS1 code for testing" default:"00800000000000"`
	Connect string `short:"c" long:"connect" description:"The validator component endpoint to" default:"http://198.13.60.39:8080"`
	RandomPrivKey []bool `short:"p" long:"random" description:"Use random private key(default key: $HOME/.sawtooth/key/$USER.priv"`
	Service string  `short:"s" long:"service" description:"Service field of NAPTR" default:"http://localhost/service.xml"`
	Regexp string  `short:"e" long:"regexp" description:"Regexp field of NAPTR" default:"!^.*$!http://example.com/cgibin/epcis!"`
	Flags rune `short:"f" long:"flags" description:"Flags field of NAPTR (default : u)" default:"117"`
	RemoveIdx uint32 `short:"r" long:"remove" description:"Index to be removed from records of GS1 code " default:"0"`
}

const action_register = "register"
const action_deregister = "deregister"
const action_add = "add"
const action_remove = "remove"
const action_get = "get"

const (
	REGISTER_GS1CODE = iota+1
	DEREGISTER_GS1CODE
	ADD_RECORD
	REMOVE_RECORD
	_
	_
	_
	_
	GET_GS1CODE_DATA
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
	user, err := user.Current()
	if is_use_random_priv_key == false {
		local_private_key, err = ioutil.ReadFile(user.HomeDir+"/.sawtooth/keys/"+user.Username+".priv")
		if err != nil {
			fmt.Println("Fail to read private key.")
			os.Exit(2)
		}
	}else{
		local_private_key = nil
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
	}

	if is_testing == true || is_verbose == true {
		fmt.Printf("command line arguments: %v\n", os.Args)
		fmt.Printf("GS1 code = %v\n", input_gs1_code)
		fmt.Printf("Service = %v\n", opts.Service)
		fmt.Printf("Regexp = %v\n", opts.Regexp)
		fmt.Printf("Flags = %c\n", opts.Flags)
		fmt.Printf("RemoveIdx = %v\n", opts.RemoveIdx)
		fmt.Printf("Test = %v\n", opts.Test)
		fmt.Printf("endpoint = %v\n", opts.Connect)
		fmt.Printf("remaining args = %v\n", args)
		fmt.Println("Username : " + user.Username)
		fmt.Println("Home Dir : " + user.HomeDir)
		fmt.Printf("transaction type = %v\n", transaction_type)
	}

	signer := MakeSigner(local_private_key, is_use_random_priv_key, is_testing || is_verbose)

	var address string
	var payload *ons_pb2.SendONSTransactionPayload
	var tr_err error
	switch transaction_type {
	case REGISTER_GS1CODE:
		payload, tr_err = MakeRegisterGS1CodePayload(input_gs1_code)
		address = MakeAddressByGS1Code(input_gs1_code)
	case DEREGISTER_GS1CODE:
		payload, tr_err = MakeDeregisterGS1CodePayload(input_gs1_code)
		address = MakeAddressByGS1Code(input_gs1_code)
	case ADD_RECORD:
		payload, tr_err = MakeAddRecordPayload(input_gs1_code, opts.Flags, opts.Service, opts.Regexp)
		address = MakeAddressByGS1Code(input_gs1_code)
	case REMOVE_RECORD:
		payload, tr_err = MakeRemoveRecordPayload(input_gs1_code, opts.RemoveIdx)
		address = MakeAddressByGS1Code(input_gs1_code)
	case GET_GS1CODE_DATA:
		address = MakeAddressByGS1Code(input_gs1_code)
		ons_query.QueryGS1CodeData(address,opts.Connect, is_verbose)
		return
	default:
		payload, tr_err = MakeRegisterGS1CodePayload(input_gs1_code)
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

func MakeSigner(priv_key_str []byte, random_priv_key bool, verify bool) (*signing.Signer) {
	context := signing.CreateContext("secp256k1")
	var private_key signing.PrivateKey
	if random_priv_key == true {
		private_key = context.NewRandomPrivateKey()
	}else{
		private_key = signing.NewSecp256k1PrivateKey(priv_key_str)
	}

	crypto_factory := signing.NewCryptoFactory(context)
	signer := crypto_factory.NewSigner(private_key)

	if verify == true {
		if random_priv_key == true {
			fmt.Printf("random private key = %v\n", private_key.AsBytes())
		}else{
			fmt.Printf("local private key  = %v\n", priv_key_str)
		}

		fmt.Printf("signer public key  = %v (%s)\n", signer.GetPublicKey().AsBytes(), signer.GetPublicKey().AsHex())
	
		message := "sawtooth ons testing program"
		signature := context.Sign([]byte(message), private_key)

		if context.Verify(signature, []byte(message), signer.GetPublicKey()) == true {
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
		log.Fatal("Failed to marshal GS1 Code data:", err)
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

func MakeRegisterGS1CodePayload(gs1_code string) (*ons_pb2.SendONSTransactionPayload, error){
	register_gs1_code_payload := &ons_pb2.SendONSTransactionPayload {
		TransactionType: ons_pb2.SendONSTransactionPayload_REGISTER_GS1CODE,
		RegisterGs1Code: &ons_pb2.SendONSTransactionPayload_RegisterGS1CodeTransactionData {
			Gs1Code : gs1_code,
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