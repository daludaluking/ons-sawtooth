package main

import (
	"log"
	"encoding/json"
	"errors"
	r "gopkg.in/gorethink/gorethink.v4"
)

const (
	GS1_CODE_TABLE = iota
	SERVICE_TYPE_TABLE
	LATEST_BLOCK_INFO
)

var g_db_name string
var g_table_names = []string {"gs1_codes", "service_types", "latest_updated_block_info"}
var g_table_map = map[string] string {g_table_names[GS1_CODE_TABLE]:"gs1Code", g_table_names[SERVICE_TYPE_TABLE]:"address", g_table_names[LATEST_BLOCK_INFO]:"block_num"}
var g_db_session *r.Session = nil

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
				log.Printf("result to create table : %#v", creation_reslut)
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
	return
}

func DBDisconnect() {
	if g_db_session != nil {
		log.Printf("database session will be closed\n")
		g_db_session.Close()
		g_db_session = nil
		g_db_name = ""
	}
}

func UpdateLatestUpdateBlockInfo() {

}

func DBInitLatestUpdatedBlockInfo() error {
	if g_db_session == nil {
		log.Printf("Not connected.\n")
		return errors.New("Not connected.")
	}

	resp, err := r.DB(g_db_name).Table(g_table_names[LATEST_BLOCK_INFO]).Insert(map[string]uint64{
		g_table_map[g_table_names[LATEST_BLOCK_INFO]]:0,}).Run(g_db_session)

	if err != nil {
		log.Printf("Failed to initialize latest updated block info table : %#v\n", err)
		return err
	}

	log.Printf("DBInitLatestUpdatedBlockInfo : %$v", resp)

	return nil
}

func DBGetLatestUpdatedBlockInfo() error {
	if g_db_session == nil {
		log.Printf("Not connected.\n")
		return errors.New("Not connected.")
	}

	cur, err := r.DB(g_db_name).Table(g_table_names[LATEST_BLOCK_INFO]).Get(g_table_map[g_table_names[LATEST_BLOCK_INFO]]).Run(g_db_session)

	if err != nil {
		log.Printf("maybe doesn't exist field : %s\n", g_table_map[g_table_names[LATEST_BLOCK_INFO]])
		return err
	}

	log.Printf("DBGetLatestUpdatedBlockInfo cursor : %#v", cur)

	var record interface{}
	err = cur.One(&record)
	if err != nil {
		log.Printf("maybe doesn't exist field : %s, error : %#v\n", g_table_map[g_table_names[LATEST_BLOCK_INFO]], err)
		return DBInitLatestUpdatedBlockInfo()
	}

	log.Printf("DBGetLatestUpdatedBlockInfo record : %#v", record)


	return nil
}
