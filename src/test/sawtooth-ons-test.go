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
	"strings"
	"net/http"
	"bytes"
	"os/user"
	"fmt"
    //"bufio"
    //"io"
    "io/ioutil"
)

var namespace = hexdigestbyString("ons")[:6]

var opts struct {
	Verbose []bool `short:"v" long:"verbose" description:"Enable verbosity"`
	GS1Code string `short:"G" long:"gs1code" description:"GS1 code for testing" default:"00800000000000"`
	Connect string `short:"C" long:"connect" description:"The validator component endpoint to" default:"http://198.13.60.39:8080/batches"`
	Test []bool `short:"T" long:"test" description:"Just for development"`
}

const action_register = "register"
const action_deregister = "deregister"
const action_add = "add"
const action_remove = "remove"

const (
	REGISTER_GS1CODE = iota
	DEREGISTER_GS1CODE
	ADD_RECORD
	REMOVE_RECORD
)

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

	if len(args) == 0 {
		fmt.Println("action is needed. (atctions = register, deregister, add, remove)")
		os.Exit(2)
	}

	input_gs1_code := opts.GS1Code
	user, err := user.Current()
	local_private_key, err := ioutil.ReadFile(user.HomeDir+"/.sawtooth/keys/"+user.Username+".priv")
	if err != nil {
		fmt.Println("Fail to read private key.")
		os.Exit(2)
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
	}

	signer := MakeSigner(local_private_key, is_testing || is_verbose)

	if is_testing == true || is_verbose == true {
		fmt.Printf("command line arguments: %v\n", os.Args)
		fmt.Printf("GS1 code = %v\n", input_gs1_code)
		fmt.Printf("Test = %v\n", opts.Test)
		fmt.Printf("endpoint = %v\n", opts.Connect)
		fmt.Printf("remaining args = %v\n", args)
		fmt.Println("Username : " + user.Username)
		fmt.Println("Home Dir : " + user.HomeDir)
		fmt.Printf("local  private key = %v\n", local_private_key)
		fmt.Printf("signer public key = %v\n", signer.GetPublicKey().AsBytes())
		if is_testing == true {
			os.Exit(0)
		}
	}

	var address string
	var payload *ons_pb2.SendONSTransactionPayload
	var tr_err error
	switch transaction_type {
	case REGISTER_GS1CODE:
		payload, tr_err = MakeRegisterGS1CodePayload(input_gs1_code)
		address = MakeAddressByGS1Code(input_gs1_code)
	case DEREGISTER_GS1CODE:
	case ADD_RECORD:
	case REMOVE_RECORD:
	default:
		payload, tr_err = MakeRegisterGS1CodePayload(input_gs1_code)
		address = MakeAddressByGS1Code(input_gs1_code)
	}

	batch_list_bytes, tr_err := MakeBatch(payload, signer, address)
	if tr_err != nil {
		log.Fatal("Failed to marshal Batch list:", err)
		return;
	}

	os.Exit(0)

	//resp, err:= http.Post("http://198.13.60.39:8080/batches", "application/octet-stream", bytes.NewBuffer(batch_list_bytes))
	resp, err:= http.Post(opts.Connect, "application/octet-stream", bytes.NewBuffer(batch_list_bytes))
	if err != nil {
		log.Fatal("Fail to send batch list", err)
		return;
	}

	fmt.Println(resp)

	defer resp.Body.Close()

	fmt.Println(resp.Body)
}

func MakeSigner(priv_key_str []byte, verify bool) (*signing.Signer) {
	context := signing.CreateContext("secp256k1")
	private_key := signing.NewSecp256k1PrivateKey(priv_key_str)
	crypto_factory := signing.NewCryptoFactory(context)
	signer := crypto_factory.NewSigner(private_key)

	if verify == true {
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

func MakeBatch(transaction_payload *ons_pb2.SendONSTransactionPayload, signer *signing.Signer, address string) ([]byte, error) {
	gs1code_reg_payload, err := proto.Marshal(transaction_payload)
	fmt.Println(transaction_payload)
	fmt.Println(gs1code_reg_payload)
	fmt.Println("address : ", address)
	
	if err != nil {
		log.Fatal("Failed to marshal GS1 Code data:", err)
		return nil, err
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
		Nonce: "",
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
	fmt.Println(transaction)
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

	fmt.Println(batch_list)
	return proto.Marshal(batch_list)
}

func MakeRegisterGS1CodePayload(gs1_code string) (*ons_pb2.SendONSTransactionPayload, error){
	register_gs1_code_payload := &ons_pb2.SendONSTransactionPayload {
		TransactionType: 0,
		RegisterGs1Code: &ons_pb2.SendONSTransactionPayload_RegisterGS1CodeTransactionData {
			Gs1Code : gs1_code,
		},
	}
	return register_gs1_code_payload, nil
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