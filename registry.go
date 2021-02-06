package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

const registryPath = "./registry.json"

func createRegistry() {
	ifErrorFatal(saveRegistry(nil))
	_, err := fmt.Fprintln(os.Stdout, "new registry created")
	if err != nil {
		// hope this never happens
		panic(err)
	}
}

func loadRegistry() (items map[int64]registryItem) {
	f, err := os.Open(registryPath)
	if err != nil {
		ifErrorPrint(err)
		createRegistry()
		return nil
	}
	defer deferWithPrintError(f.Close)
	if err = json.NewDecoder(f).Decode(&items); err != nil {
		ifErrorPrint(err)
		createRegistry()
		return nil
	}
	return
}

func saveRegistry(items map[int64]registryItem) error {
	f, err := os.Create(registryPath)
	ifErrorFatal(err)
	defer deferWithPrintError(f.Close)
	encoder := json.NewEncoder(f)
	return encoder.Encode(items)
}

func addToRegistry(registry map[int64]registryItem, allTypes []itemType, newItem string) (map[int64]registryItem, string, error) {
	var (
		id    int64
		price float64
	)
	_, err := fmt.Sscanf(newItem, "%d %f", &id, &price)
	if err != nil {
		return registry, "", err
	}
	typeName := getItemName(allTypes, id)
	if typeName == "" {
		return registry, "", errors.New("cannot resolve item by ID")
	}
	if registry == nil {
		registry = make(map[int64]registryItem)
	}
	registry[id] = registryItem{
		TypeId:   id,
		Price:    price,
		TypeName: typeName,
	}
	return registry, fmt.Sprintf("added %s\n", typeName), saveRegistry(registry)
}
