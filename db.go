package main

import "github.com/hashicorp/go-memdb"

const (
	tableContracts = "contracts"
	indexContracts = "Id"
)

func connectToDatabase() (*memdb.MemDB, error) {
	schema := &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			tableContracts: {
				Name: tableContracts,
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.IntFieldIndex{Field: indexContracts},
					},
				},
			},
		},
	}
	return memdb.NewMemDB(schema)
}

func isExistsContractCheckDB(db *memdb.MemDB, contract contract) (exists bool) {
	txn := db.Txn(false)
	e, err := txn.First(tableContracts, "id", contract.Id)
	ifErrorFatal(err)
	txn.Commit()
	return e != nil
}

func isNewlyCreatedContractCheckDB(db *memdb.MemDB, contract contract) (inserted bool) {
	if isExistsContractCheckDB(db, contract) {
		return false
	}
	txn := db.Txn(true)
	ifErrorFatal(txn.Insert(tableContracts, contract))
	txn.Commit()
	return true
}
