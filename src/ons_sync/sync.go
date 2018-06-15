package main

import (
	"log"
	"errors"
	"strings"
	"sync"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	//"protobuf/ons_pb2"
	r "gopkg.in/gorethink/gorethink.v4"
)

const (
	GS1_CODE_TABLE = iota
	SERVICE_TYPE_TABLE
	LATEST_BLOCK_INFO
	NONE
)

var g_db_name string
var g_table_names = []string {"gs1_codes", "service_types", "latest_updated_block_info"}
var g_table_map = map[string] string {g_table_names[GS1_CODE_TABLE]:"Gs1Code", g_table_names[SERVICE_TYPE_TABLE]:"Address", g_table_names[LATEST_BLOCK_INFO]:"index"}
var g_db_session *r.Session = nil
var g_latest_block_id string = "0000000000000000"
var g_latest_block_num float64 = 0
var g_mutex *sync.Mutex = nil

func prettyPrint(v interface{}) {
	buf, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		log.Printf("PrintPrettyJson : json.MarshalIndent : error %v\n", err);
		return
	}

	buf = append(buf, '\n')
	log.Printf(string(buf))
}

func DBConnect(url string, db_name string, verbose bool) {
	log.Printf("Connect %s\n", url)
	session, err := r.Connect(r.ConnectOpts{
		Address: url,
	})

	if err != nil {
		log.Printf("Failed to connect %s\n", url)
		log.Fatalln(err)
	}

	resp, err := r.DBList().Contains(db_name).Run(session)
	if err != nil {
		log.Printf("Failed to check %s database exist\n", db_name)
		log.Fatalln(err)
	}

	var contained bool
	err = resp.One(&contained)
	if err != nil {
		log.Printf("Failed to retrieves the first document from the result set to check %s database exist\n", db_name)
		log.Fatalln(err)
	}

	if verbose == true {
		log.Printf("%s database is exist : %#v\n", db_name, contained)
	}

	if contained == false {
		resp, err = r.DBCreate(db_name).Run(session)
		if err != nil {
			log.Fatalln(err)
		}
		var rows []map[string]interface{}
		err = resp.All(&rows)
		if err != nil {
			log.Fatalln(err)
		}

		if int(rows[0]["dbs_created"].(float64)) == 1 {
			log.Printf("%s database is created\n", db_name)
		}

		if verbose == true {
			prettyPrint(rows)
		}
	}

	for table_name, pk_name := range g_table_map {
		resp, err = r.DB(db_name).TableList().Contains(table_name).Run(session)
		var table_contained bool
		err = resp.One(&table_contained)
		if err != nil {
			log.Printf("Failed to retrieves the first document from the result set to check %s database exist\n", db_name)
			log.Fatalln(err)
		}

		if verbose == true {
			log.Printf("is %s table exist in %s database : %#v\n", table_name, db_name, table_contained)
		}

		if table_contained == false {
			resp, err = r.DB(db_name).TableCreate(table_name, r.TableCreateOpts{PrimaryKey:pk_name,}).Run(session)
			var creation_reslut map[string] interface{}
			err = resp.One(&creation_reslut)

			if verbose == true {
				log.Printf("result to create table : %#v", creation_reslut["tables_created"])
			}

			if creation_reslut["tables_created"].(float64) == 1 {
				log.Printf("Is table creation succeeded : %#v", creation_reslut["tables_created"].(float64))
			}

		}else{
			log.Printf("%s table is exist in %s\n", table_name, db_name)
		}
	}
	g_db_name = db_name
	g_db_session = session
	g_mutex = &sync.Mutex{}
	return
}

func DBDisconnect() {
	if g_db_session != nil {
		log.Printf("database session will be closed\n")
		g_db_session.Close()
		g_db_session = nil
		g_db_name = ""
		g_mutex = nil
	}
}

func UpdateLatestUpdateBlockInfo() {

}

func checkNormalOPResult(cur *r.Cursor, op string, func_name string) bool {
		//for test
	var results map[string]float64
	err := cur.One(&results)
	if err != nil {
		log.Printf("Failed to checkNormalOPResult(func : %s, op : %s) : %v\n", func_name, op, err)
	}

	log.Printf("%s : %s result: %v", func_name, op, results)

	if results[op] == 1 {
		return true
	}

	return false
}

func DBInitLatestUpdatedBlockInfo(verbose bool) (float64, error) {
	if g_db_session == nil {
		log.Printf("Not connected.\n")
		return -1, errors.New("Not connected.")
	}

	cur, err := r.DB(g_db_name).Table(g_table_names[LATEST_BLOCK_INFO]).Insert(map[string]interface{}{
		g_table_map[g_table_names[LATEST_BLOCK_INFO]]:0,
		"block_num": 0,
		"block_id": "0000000000000000",
		"previous_block_id": "0000000000000000", //the number of zero is important.
		}).Run(g_db_session)
	if err != nil {
		log.Printf("Failed to initialize latest updated block info table : %#v\n", err)
		return -1, err
	}
	defer cur.Close()
	checkNormalOPResult(cur, "inserted", "DBInitLatestUpdatedBlockInfo")

	g_latest_block_num = 0
	return g_latest_block_num, nil
}

func DBUpdateLatestUpdatedBlockInfo(block_num float64, block_id string, prev_block_id string) error {
	if g_db_session == nil {
		log.Printf("Not connected.\n")
		return errors.New("Not connected.")
	}

	cur, err := r.DB(g_db_name).Table(g_table_names[LATEST_BLOCK_INFO]).Update(map[string]interface{}{
		g_table_map[g_table_names[LATEST_BLOCK_INFO]]:0,
		"block_num": block_num,
		"block_id": block_id,
		"previous_block_id": prev_block_id, //the number of zero is important.
		}).Run(g_db_session)
	if err != nil {
		log.Printf("Failed to initialize latest updated block info table : %#v\n", err)
		return err
	}
	defer cur.Close()
	checkNormalOPResult(cur, "replaced", "DBUpdateLatestUpdatedBlockInfo")

	return nil
}

func DBGetLatestUpdatedBlockInfo(verbose bool) (float64, error) {
	if g_db_session == nil {
		log.Printf("Not connected.\n")
		return -1, errors.New("Not connected.")
	}

	//lastest updated block info table은 항상 index 0인 record만 사용한다.
	cur, err := r.DB(g_db_name).Table(g_table_names[LATEST_BLOCK_INFO]).Get(0).Run(g_db_session)
	if err != nil {
		log.Printf("maybe doesn't exist field : %s\n", g_table_map[g_table_names[LATEST_BLOCK_INFO]])
		return -1, err
	}
	defer cur.Close()

	var record map[string]interface{}
	err = cur.One(&record)
	if err != nil {
		log.Printf("maybe doesn't exist field : %s, error : %#v\n", g_table_map[g_table_names[LATEST_BLOCK_INFO]], err)
		return DBInitLatestUpdatedBlockInfo(verbose)
	}

	if verbose == true {
		log.Printf("DBGetLatestUpdatedBlockInfo\n - latest block num : %#v", record["block_num"].(float64))
		log.Printf(" - latest block id : %#v", record["block_id"].(string))
	}

	g_latest_block_num = record["block_num"].(float64)
	g_latest_block_id = record["previous_block_id"].(string)

	return g_latest_block_num, nil
}

func DBGetLatestUpdatedBlock() (float64, string) {
	return g_latest_block_num, g_latest_block_id
}

func DBUpdateOrInsert(table_idx int, pk_v string, block_num float64, v interface{}) error {

	if g_db_session == nil {
		log.Printf("Not connected.\n")
		return errors.New("Not connected.")
	}

	if len(g_table_names) < table_idx {
		log.Printf("Invalid table index : %d.\n", table_idx)
		return errors.New("Invalid table index")
	}

	g_mutex.Lock()
	defer g_mutex.Unlock()

	table_name := g_table_names[table_idx]

	//lastest updated block info table은 항상 index 0인 record만 사용한다.
	cur, err := r.DB(g_db_name).Table(table_name).Get(pk_v).Run(g_db_session)
	if err != nil {
		log.Printf("maybe %s doesn't exist field : %v\n", g_table_map[g_table_names[GS1_CODE_TABLE]], err)
		return err
	}
	defer cur.Close()

	var record map[string]interface{}
	//var record ons_pb2.GS1CodeData
	err = cur.One(&record)
	if err != nil {
		log.Printf("DBUpdateOrInsert : insert item because %s item doesn't exist field (2): %v\n", g_table_map[g_table_names[GS1_CODE_TABLE]], err)
		cur, err := r.DB(g_db_name).Table(table_name).Insert(v).Run(g_db_session)
		if err != nil {
			log.Printf("Failed to item : %#v\n", err)
			return err
		}
		defer cur.Close()
		checkNormalOPResult(cur, "inserted", "DBUpdateOrInsert")
	}else{
		//if old data, skip..
		if block_num <= record["BlockNum"].(float64) {
			log.Printf("skip item because of old block data : %s, %s, %v\n", g_table_map[g_table_names[GS1_CODE_TABLE]], pk_v, block_num)
			return nil
		}

		log.Printf("DBUpdateOrInsert : update item because %s item exist(%s)", g_table_map[g_table_names[GS1_CODE_TABLE]], pk_v)
		cur, err := r.DB(g_db_name).Table(table_name).Update(v).Run(g_db_session)
		if err != nil {
			log.Printf("Failed to update item : %v\n", err)
			return err
		}
		defer cur.Close()
		checkNormalOPResult(cur, "replaced", "DBUpdateOrInsert")
	}

	return nil
}

func DBDeleteAddress(address string) error {
	if g_db_session == nil {
		log.Printf("Not connected.\n")
		return errors.New("Not connected.")
	}

	//lastest updated block info table은 항상 index 0인 record만 사용한다.
	cur, err := r.DB(g_db_name).Table(g_table_names[GS1_CODE_TABLE]).Filter(map[string]string{
		"Address":address,
	}).Run(g_db_session)
	if err != nil {
		return err
	}
	defer cur.Close()

	if checkNormalOPResult(cur, "deleted", "DBDeleteAddress in "+g_table_names[GS1_CODE_TABLE]) == false {
		cur, err := r.DB(g_db_name).Table(g_table_names[SERVICE_TYPE_TABLE]).Filter(map[string]string{
			"Address":address,
		}).Run(g_db_session)
		defer cur.Close()
		if err != nil {
			return err
		}
		if checkNormalOPResult(cur, "deleted", "DBDeleteAddress in "+g_table_names[SERVICE_TYPE_TABLE]) == false {
			log.Printf("Nothing has been deleted.")
		}
	}
	return nil
}

func hexdigest(str string) string {
	hash := sha512.New()
	hash.Write([]byte(str))
	hashBytes := hash.Sum(nil)
	return strings.ToLower(hex.EncodeToString(hashBytes))
}

func GetTableIdxByAddress(address string) int{
	namespace := hexdigest("ons")[:6]

	target_address := namespace + hexdigest("gs1")[:8]
	log.Printf("GetTableIdxByAddress : %s : %s\n", address, target_address)
	if address[:14] == target_address {
		return GS1_CODE_TABLE
	}

	target_address = namespace + hexdigest("service-type")[:8]
	log.Printf("GetTableIdxByAddress : %s : %s\n", address, target_address)
	if address[:14] == target_address {
		return SERVICE_TYPE_TABLE
	}

	return NONE
}
