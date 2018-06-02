package main

import (
	//flags "github.com/jessevdk/go-flags"
	//"os"
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
	//"syscall"
	"fmt"
)

var namespace = hexdigestbyString("ons")[:6]

func main() {
	context := signing.CreateContext("secp256k1")
	private_key := context.NewRandomPrivateKey()
	crypto_factory := signing.NewCryptoFactory(context)
	signer := crypto_factory.NewSigner(private_key)

	gs1code_data := &ons_pb2.SendONSTransactionPayload {
		TransactionType: 0,
		RegisterGs1Code: &ons_pb2.SendONSTransactionPayload_RegisterGS1CodeTransactionData {
			Gs1Code : "00800000000000",
		},
	}

	gs1code_reg_payload, err := proto.Marshal(gs1code_data)
	fmt.Println(gs1code_data)
	fmt.Println(gs1code_reg_payload)
	if err != nil {
		log.Fatal("Failed to marshal GS1 Code data:", err)
		return;
	}

	address := namespace + hexdigestbyString(gs1code_data.RegisterGs1Code.Gs1Code)[:64]
	//address := namespace + hexdigestbyString("Testing")
	fmt.Println("address : ", address)
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
		return;
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
		return;
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
	batch_list_bytes, err := proto.Marshal(batch_list)
	if err != nil {
		log.Fatal("Failed to marshal Batch list:", err)
		return;
	}

	resp, err:= http.Post("http://198.13.60.39:8080/batches", "application/octet-stream", bytes.NewBuffer(batch_list_bytes))
	if err != nil {
		log.Fatal("Fail to send batch list", err)
		return;
	}

	fmt.Println(resp)

	defer resp.Body.Close()

	fmt.Println(resp.Body)
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
