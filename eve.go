package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	host = "esi.evetech.net"

	eveServer    = "tranquility"
	regionIdJita = "10000002"

	apiContracts     = "/latest/contracts/public/"
	apiContractItems = "/latest/contracts/public/items/"

	ifNoneMatchHeader = "If-None-Match"
	typesUrl          = "https://eve-files.com/chribba/typeid.txt"
)

func checkResponse(resp *http.Response) error {
	if resp.StatusCode == 404 {
		return io.EOF
	}
	if resp.StatusCode >= 400 {
		return errors.New(resp.Status)
	}
	return nil
}

func executeQuery(client httpClient, path, itemId string, page int, data interface{}, eTagsA ...*sync.Map) error {
	var (
		etag             = ""
		eTags  *sync.Map = nil
		key              = fmt.Sprintf("%s%s?%d", path, itemId, page)
		newUrl           = url.URL{
			Scheme: "https",
			Host:   host,
			Path:   path + itemId,
		}
	)
	if c := len(eTagsA); c > 0 {
		if c != 1 {
			return errors.New("only one eTags argument expected")
		}
		if eTags = eTagsA[0]; eTags != nil {
			if etagI, ok := eTags.Load(key); ok {
				etag = etagI.(string)
			}
		}
	}
	newUrl.RawQuery = url.Values{
		"datasource": []string{eveServer},
		"page":       []string{strconv.Itoa(page)},
	}.Encode()
	req, err := http.NewRequest(http.MethodGet, newUrl.String(), nil)
	ifErrorFatal(err)
	// this header will save time on pages that have not been changed
	req.Header.Add(ifNoneMatchHeader, etag)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer deferWithPrintError(resp.Body.Close)
	if err := checkResponse(resp); err != nil {
		return err
	}
	if resp.StatusCode == 200 {
		if eTags != nil {
			eTags.Store(key, resp.Header.Get("etag"))
		}
		dec := json.NewDecoder(resp.Body)
		return dec.Decode(&data)
	}
	// there is no point in looking for data in an response other than status 200
	return nil
}

type eveConnector struct {
	client httpClient
}

var eTags sync.Map

func (c *eveConnector) getContracts(regionId string, page int) (data []contract, err error) {
	var client httpClient = http.DefaultClient
	if c.client != nil {
		client = c.client
	}
	if err = executeQuery(client, apiContracts, regionId, page, &data, &eTags); err != nil {
		return nil, err
	}
	return data, nil
}

func (c *eveConnector) getContractItems(contractId string, page int) (data []contractItem, err error) {
	var client httpClient = http.DefaultClient
	if c.client != nil {
		client = c.client
	}
	if err = executeQuery(client, apiContractItems, contractId, page, &data, &eTags); err != nil {
		return nil, err
	}
	return data, nil
}

func getItemTypeFromString(s string) (i itemType, err error) {
	if f := strings.Fields(s); len(f) > 0 {
		var id int64
		id, err = strconv.ParseInt(f[0], 10, 64)
		if err == nil {
			i = itemType{
				typeId:   id,
				typeName: strings.TrimSpace(strings.Join(f[1:], " ")),
			}
		}
		return
	}
	err = errors.New("format error")
	return
}

func loadEVEItemTypes(client httpClient) (items []itemType, err error) {
	req, err := http.NewRequest(http.MethodGet, typesUrl, nil)
	ifErrorFatal(err)
	var resp *http.Response
	if resp, err = client.Do(req); err != nil {
		return nil, err
	}
	defer deferWithPrintError(resp.Body.Close)
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		if i, e := getItemTypeFromString(scanner.Text()); e == nil {
			items = append(items, i)
		}
	}
	return
}

func (c *eveConnector) getItemTypes() []itemType {
	var client httpClient = http.DefaultClient
	if c.client != nil {
		client = c.client
	}
	items, err := loadEVEItemTypes(client)
	ifErrorFatal(err)
	return items
}

func loadAllContracts(
	eve eveConnector,
	regionId string,
	logger io.Writer,
	conCh chan<- contract,
	errCh chan<- error,
) {
	defer close(conCh)
	defer close(errCh)
	var page = 1
	for {
		fmt.Fprintf(logger, "processing page %d...\n", page)
		data, err := eve.getContracts(regionId, page)
		if err != nil {
			if err == io.EOF {
				break
			}
			errCh <- err
			<-time.After(time.Second * 5)
		}
		for _, contract := range data {
			conCh <- contract
		}
		fmt.Fprintf(logger, "got %d new contracts\n", len(data))
		page++
	}
}

func loadContractItems(eve eveConnector, contractId int64) ([]contractItem, error) {
	var (
		page  = 1
		items = make([]contractItem, 0, 2)
	)
	for {
		i, err := eve.getContractItems(strconv.FormatInt(contractId, 10), page)
		if err == io.EOF {
			break
		}
		if err != nil {
			return items, err
		}
		items = append(items, i...)
		page++
	}
	return items, nil
}

func checkSuitable(registry map[int64]registryItem, contract contract, items []contractItem) bool {
	var (
		excluded              = false
		contractBound float64 = 0
	)
	for _, item := range items {
		if curr, ok := registry[item.TypeId]; ok {
			var r = item.Runs
			if r < 0 {
				r = math.MaxInt16 // this is the original blueprint, it is very valuable
			}
			contractBound += curr.Price * float64(r*item.Quantity)
		}
		if !item.IsIncluded {
			excluded = true
			break
		}
	}
	return !excluded && contractBound > 0 && (contract.Price/1000000)-contractBound < 0.001
}

func monCheckContract(contract contract, eve eveConnector, registry map[int64]registryItem, chSignal chan<- registrySignal, logger io.Writer) {
	items, err := loadContractItems(eve, contract.Id)
	if err != nil {
		ifErrorPrint(err)
		return
	}
	if checkSuitable(registry, contract, items) {
		fmt.Fprintln(logger, "FOUND")
		chSignal <- registrySignal{
			contract: contract,
			items:    items,
		}
	}
}
