package main

import (
	"fmt"
	"net/http"
	"time"
)

type (
	contract struct {
		Id                  int64     `json:"contract_id"`
		Buyout              float64   `json:"buyout"`     // Auction
		Collateral          float64   `json:"collateral"` // Couriers
		Price               float64   `json:"price"`      // ItemExchange
		DateExpired         time.Time `json:"date_expired"`
		DateIssued          time.Time `json:"date_issued"`
		ForCorporation      bool      `json:"for_corporation"`
		IssuerCorporationId int64     `json:"issuer_corporation_id"`
		Title               string    `json:"title"`
		Type                string    `json:"type"` // [ unknown, item_exchange, auction, courier, loan ]
		Volume              float64   `json:"volume"`
	}
	contractItem struct {
		RecordId int64 `json:"record_id"`
		// ItemId             *int64 `json:"item_id"`
		IsBlueprintCopy    bool  `json:"is_blueprint_copy"`
		IsIncluded         bool  `json:"is_included"`
		MaterialEfficiency int32 `json:"material_efficiency"`
		TimeEfficiency     int32 `json:"time_efficiency"`
		Runs               int32 `json:"runs"`
		Quantity           int32 `json:"quantity"`
		TypeId             int64 `json:"type_id"`
	}
	registrySignal struct {
		contract contract
		items    []contractItem
	}
	registryItem struct {
		TypeId   int64   `json:"type_id"`
		Price    float64 `json:"price"`
		TypeName string  `json:"type_name"`
	}
	itemType struct {
		typeId   int64
		typeName string
	}

	httpClient interface {
		Do(*http.Request) (*http.Response, error)
	}
)

const (
	itemExchange = "item_exchange"
)

func isPublicItemExchangeContract(contract contract) bool {
	return !contract.ForCorporation && contract.Type == itemExchange
}

func getItemName(items []itemType, id int64) string {
	for _, i := range items {
		if i.typeId == id {
			return fmt.Sprintf("%d, %s", i.typeId, i.typeName)
		}
	}
	return fmt.Sprintf("unknown ID %d", id)
}
