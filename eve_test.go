package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

type (
	httpRequested struct {
		url    string
		header http.Header
	}
	httpClientTest struct {
		data     string
		requests []httpRequested
	}
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func (h *httpClientTest) Do(r *http.Request) (*http.Response, error) {
	h.requests = append(h.requests, httpRequested{
		url:    r.URL.String(),
		header: r.Header.Clone(),
	})
	resp := http.Response{
		Status:           "200 OK",
		StatusCode:       200,
		Proto:            "https",
		Header:           nil,
		Body:             ioutil.NopCloser(strings.NewReader(h.data)),
		ContentLength:    int64(len(h.data)),
		TransferEncoding: nil,
		Close:            false,
		Uncompressed:     false,
		Trailer:          nil,
		Request:          r,
		TLS:              nil,
	}
	return &resp, nil
}

func makeHttpClientTest(stringData string) httpClientTest {
	return httpClientTest{
		data: stringData,
	}
}

func Test_executeQuery(t *testing.T) {
	type args struct {
		client httpClientTest
		path   string
		itemId string
		page   int
		data   interface{}
	}
	makeDataReceiver := func() interface{} {
		var r map[string]interface{}
		return &r
	}
	checkResponse := func(a args) error {
		if len(a.client.requests) != 1 {
			return fmt.Errorf("expected one request, got %d", len(a.client.requests))
		}
		u, err := url.Parse(a.client.requests[0].url)
		if err != nil {
			return err
		}
		if u.Path != a.path+a.itemId {
			return fmt.Errorf("unexpected URI: %s", a.client.requests[0].url)
		}
		if gotPage := u.Query().Get("page"); gotPage != strconv.Itoa(a.page) {
			return fmt.Errorf("unexpected page: want %d, got %s", a.page, gotPage)
		}
		return nil
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test-1",
			args: args{
				client: makeHttpClientTest("{}"),
				path:   "/somePath/",
				itemId: strconv.Itoa(rand.Int()),
				page:   rand.Int(),
				data:   makeDataReceiver(),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := executeQuery(&tt.args.client, tt.args.path, tt.args.itemId, tt.args.page, tt.args.data); (err != nil) != tt.wantErr {
				t.Errorf("executeQuery() error = %v, wantErr %v", err, tt.wantErr)
			} else if err == nil {
				if err = checkResponse(tt.args); err != nil {
					t.Error(err)
				}
			}
		})
	}
}

func Test_checkResponse(t *testing.T) {
	type args struct {
		resp *http.Response
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "200 OK",
			args: args{
				resp: &http.Response{
					Status:     "200",
					StatusCode: http.StatusOK,
					Proto:      "https",
				},
			},
			wantErr: false,
		},
		{
			name: "204 No Countent",
			args: args{
				resp: &http.Response{
					Status:     "204",
					StatusCode: http.StatusNoContent,
					Proto:      "https",
				},
			},
			wantErr: false,
		},
		{
			name: "304 Not Modified",
			args: args{
				resp: &http.Response{
					Status:     "304",
					StatusCode: http.StatusNotModified,
					Proto:      "https",
				},
			},
			wantErr: false,
		},
		{
			name: "404 Not Found",
			args: args{
				resp: &http.Response{
					Status:     "404",
					StatusCode: http.StatusNotFound,
					Proto:      "https",
				},
			},
			wantErr: true,
		},
		{
			name: "500 Internal Server Error",
			args: args{
				resp: &http.Response{
					Status:     "500",
					StatusCode: http.StatusInternalServerError,
					Proto:      "https",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := checkResponse(tt.args.resp); (err != nil) != tt.wantErr {
				t.Errorf("checkResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_getItemTypeFromString(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name    string
		args    args
		wantI   itemType
		wantErr bool
	}{
		{
			name: "positive",
			args: args{s: "1234567    Some Item Name"},
			wantI: itemType{
				typeId:   1234567,
				typeName: "Some Item Name",
			},
			wantErr: false,
		},
		{
			name: "check spaces",
			args: args{s: " 1234567    Some Item Name "},
			wantI: itemType{
				typeId:   1234567,
				typeName: "Some Item Name",
			},
			wantErr: false,
		},
		{
			name:    "wrong int format",
			args:    args{s: "1234567Some Item Name "},
			wantErr: true,
		},
		{
			name:    "wrong line format",
			args:    args{s: "------------------------"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotI, err := getItemTypeFromString(tt.args.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("getItemTypeFromString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotI, tt.wantI) {
				t.Errorf("getItemTypeFromString() gotI = %v, want %v", gotI, tt.wantI)
			}
		})
	}
}

func Test_loadEVEItemTypes(t *testing.T) {
	tests := []struct {
		name      string
		client    httpClientTest
		wantItems []itemType
		wantErr   bool
	}{
		{
			name:   "test foo bar",
			client: makeHttpClientTest("123 Foo\n321 Bar"),
			wantItems: []itemType{
				{typeId: 123, typeName: "Foo"},
				{typeId: 321, typeName: "Bar"},
			},
			wantErr: false,
		},
	}
	checkResponse := func(c httpClientTest) error {
		if len(c.requests) != 1 {
			return errors.New("wrong requests count")
		}
		if c.requests[0].url != typesUrl {
			return errors.New("wrong request url")
		}
		return nil
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotItems, err := loadEVEItemTypes(&tt.client)
			if (err != nil) != tt.wantErr {
				t.Errorf("loadEVEItemTypes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				if err = checkResponse(tt.client); err != nil {
					t.Error(err)
				} else {
					if !reflect.DeepEqual(gotItems, tt.wantItems) {
						t.Errorf("loadEVEItemTypes() gotItems = %v, want %v", gotItems, tt.wantItems)
					}
				}
			}
		})
	}
}
