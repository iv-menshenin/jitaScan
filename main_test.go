package main

import (
	"bytes"
	"math/rand"
	"testing"
)

func Test_checkSuitable(t *testing.T) {
	type args struct {
		registry map[int64]registryItem
		contract contract
		items    []contractItem
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "found",
			args: args{
				registry: map[int64]registryItem{
					123: {
						TypeId:   123,
						Price:    40.99,
						TypeName: "Foo",
					},
					321: {
						TypeId:   321,
						Price:    899.50,
						TypeName: "Bar",
					},
				},
				contract: contract{
					Id:             144,
					Price:          880000000.5,
					ForCorporation: false,
					Title:          "Test",
					Type:           itemExchange,
				},
				items: []contractItem{
					{
						RecordId:        1,
						IsBlueprintCopy: true,
						IsIncluded:      true,
						Runs:            1,
						Quantity:        1,
						TypeId:          321,
					},
				},
			},
			want: true,
		},
		{
			name: "not found",
			args: args{
				registry: map[int64]registryItem{
					123: {
						TypeId:   123,
						Price:    40.99,
						TypeName: "Foo",
					},
					321: {
						TypeId:   321,
						Price:    899.50,
						TypeName: "Bar",
					},
				},
				contract: contract{
					Id:             144,
					Price:          880000000.5,
					ForCorporation: false,
					Title:          "Test",
					Type:           itemExchange,
				},
				items: []contractItem{
					{
						RecordId:        2,
						IsBlueprintCopy: true,
						IsIncluded:      true,
						Runs:            1,
						Quantity:        1,
						TypeId:          122,
					},
				},
			},
			want: false,
		},
		{
			name: "multicount",
			args: args{
				registry: map[int64]registryItem{
					123: {
						TypeId:   123,
						Price:    40.99,
						TypeName: "Foo",
					},
					321: {
						TypeId:   321,
						Price:    899.50,
						TypeName: "Bar",
					},
				},
				contract: contract{
					Id:             144,
					Price:          1880000000.88, // 1880.98,
					ForCorporation: false,
					Title:          "Test",
					Type:           itemExchange,
				},
				items: []contractItem{
					{
						RecordId:        1,
						IsBlueprintCopy: true,
						IsIncluded:      true,
						Runs:            1,
						Quantity:        2,
						TypeId:          123,
					},
					{
						RecordId:        2,
						IsBlueprintCopy: true,
						IsIncluded:      true,
						Runs:            2,
						Quantity:        1,
						TypeId:          111,
					},
					{
						RecordId:        3,
						IsBlueprintCopy: true,
						IsIncluded:      true,
						Runs:            2,
						Quantity:        1,
						TypeId:          321,
					},
				},
			},
			want: true,
		},
		{
			name: "price too big",
			args: args{
				registry: map[int64]registryItem{
					123: {
						TypeId:   123,
						Price:    40.99,
						TypeName: "Foo",
					},
					321: {
						TypeId:   321,
						Price:    899.50,
						TypeName: "Bar",
					},
				},
				contract: contract{
					Id:             144,
					Price:          1881000000,
					ForCorporation: false,
					Title:          "Test",
					Type:           itemExchange,
				},
				items: []contractItem{
					{
						RecordId:        2,
						IsBlueprintCopy: true,
						IsIncluded:      true,
						Runs:            1,
						Quantity:        2,
						TypeId:          123,
					},
					{
						RecordId:        3,
						IsBlueprintCopy: true,
						IsIncluded:      true,
						Runs:            2,
						Quantity:        1,
						TypeId:          321,
					},
				},
			},
			want: false,
		},
		{
			name: "with original",
			args: args{
				registry: map[int64]registryItem{
					123: {
						TypeId:   123,
						Price:    40.99,
						TypeName: "Foo",
					},
					321: {
						TypeId:   321,
						Price:    899.50,
						TypeName: "Bar",
					},
				},
				contract: contract{
					Id:             144,
					Price:          18900000000, // 1880.98,
					ForCorporation: false,
					Title:          "Test",
					Type:           itemExchange,
				},
				items: []contractItem{
					{
						RecordId:        2,
						IsBlueprintCopy: true,
						IsIncluded:      true,
						Runs:            -1,
						Quantity:        2,
						TypeId:          123,
					},
					{
						RecordId:        3,
						IsBlueprintCopy: true,
						IsIncluded:      true,
						Runs:            2,
						Quantity:        1,
						TypeId:          321,
					},
				},
			},
			want: true,
		},
		{
			name: "excluded",
			args: args{
				registry: map[int64]registryItem{
					123: {
						TypeId:   123,
						Price:    40.99,
						TypeName: "Foo",
					},
					321: {
						TypeId:   321,
						Price:    899.50,
						TypeName: "Bar",
					},
				},
				contract: contract{
					Id:             144,
					Price:          190000000, // 1880.98,
					ForCorporation: false,
					Title:          "Test",
					Type:           itemExchange,
				},
				items: []contractItem{
					{
						RecordId:        2,
						IsBlueprintCopy: true,
						IsIncluded:      true,
						Runs:            1,
						Quantity:        2,
						TypeId:          123,
					},
					{
						RecordId:        3,
						IsBlueprintCopy: true,
						IsIncluded:      false,
						Runs:            2,
						Quantity:        1,
						TypeId:          333,
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkSuitable(tt.args.registry, tt.args.contract, tt.args.items)
			if got != tt.want {
				t.Errorf("checkSuitable() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Benchmark_checkSuitable(b *testing.B) {
	type args struct {
		registry map[int64]registryItem
		contract contract
		items    []contractItem
	}
	var a = args{
		registry: nil,
		contract: contract{
			Id:             rand.Int63(),
			Price:          rand.Float64(),
			ForCorporation: false,
			Type:           itemExchange,
		},
		items: nil,
	}
	a.registry = make(map[int64]registryItem, 50)
	for i := 0; i < rand.Intn(50)+10; i++ {
		var typeId = rand.Int63()
		a.registry[typeId] = registryItem{
			TypeId:   typeId,
			Price:    rand.Float64(),
			TypeName: "",
		}
	}
	a.items = make([]contractItem, 0, rand.Intn(10)+3)
	for i := 0; i < cap(a.items); i++ {
		var typeId = rand.Int63()
		if rand.Intn(100) < 8 {
			for k := range a.registry {
				typeId = k
				break
			}
		}
		var item = contractItem{
			RecordId:        int64(i),
			IsBlueprintCopy: true,
			IsIncluded:      true,
			Runs:            int32(rand.Intn(4) + 1),
			Quantity:        1,
			TypeId:          typeId,
		}
		a.items = append(a.items, item)
	}
	b.ResetTimer()
	for bi := 0; bi < b.N; bi++ {
		checkSuitable(a.registry, a.contract, a.items)
	}
}

func Test_alerter(t *testing.T) {
	type args struct {
		registry map[int64]registryItem
		chSignal chan registrySignal
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "test printout",
			args: args{
				registry: map[int64]registryItem{
					123: {
						TypeId:   123,
						Price:    100,
						TypeName: "TESTITEM",
					},
				},
				chSignal: make(chan registrySignal),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var w = bytes.NewBuffer([]byte{})
			go alerter(w, tt.args.registry, tt.args.chSignal)
			for i := 0; i < 5; i++ {
				tt.args.chSignal <- registrySignal{
					contract: contract{
						Id:             rand.Int63(),
						ForCorporation: false,
						Title:          "Test",
						Type:           itemExchange,
					},
					items: []contractItem{
						{
							RecordId:        1,
							IsBlueprintCopy: true,
							IsIncluded:      true,
							Runs:            1,
							Quantity:        1,
							TypeId:          123,
						},
					},
				}
			}
			close(tt.args.chSignal)
			if w.String() == "" {
				t.Error("output is empty")
			} else {
				println(w.String())
			}
		})
	}
}
