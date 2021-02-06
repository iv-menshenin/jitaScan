package main

import (
	"flag"
	"fmt"
	"github.com/gordonklaus/portaudio"
	"github.com/hashicorp/go-memdb"
	"io"
	"io/ioutil"
	"os"
	"time"
)

const (
	paramVerbose = "verbose"
	paramRegion  = "region"
	paramShow    = "show"
	paramTypes   = "types"
	paramAdd     = "add"

	commandMonitoring = "monitoring"
	commandRegistry   = "registry"
)

type (
	monitoringState struct {
		db      *memdb.MemDB
		verbose bool
		region  string
		logger  io.Writer
	}
	registryState struct {
		showRegistry bool
		showTypes    bool
		addItem      string
		// registry     map[int64]registryItem
	}
	programState struct {
		monitoring monitoringState
		registry   registryState
		eve        eveConnector
		output     io.Writer
	}
)

func ifErrorPrint(err error) {
	if err != nil {
		ifErrorFatal2(fmt.Fprintf(os.Stderr, "%s ERROR: %s\n", time.Now().Format(time.RFC3339), err))
	}
}

func ifErrorFatal(err error) {
	if err != nil {
		ifErrorFatal2(fmt.Fprintf(os.Stderr, "%s ERROR: %s\n", time.Now().Format(time.RFC3339), err))
		os.Exit(1)
	}
}

func ifErrorFatal2(_ int, err error) {
	if err != nil {
		_, err := fmt.Fprintf(os.Stderr, "%s ERROR: %s\n", time.Now().Format(time.RFC3339), err)
		if err != nil {
			panic(err)
		}
		os.Exit(1)
	}
}

func (state *programState) init() map[string]*flag.FlagSet {
	state.output = os.Stdout

	fsMonitoring := flag.NewFlagSet(commandMonitoring, flag.PanicOnError)
	fsMonitoring.BoolVar(&state.monitoring.verbose, paramVerbose, false, "show information messages")
	fsMonitoring.StringVar(&state.monitoring.region, paramRegion, regionIdJita, "select a region to search for contracts")

	fsRegistry := flag.NewFlagSet(commandRegistry, flag.PanicOnError)
	fsRegistry.BoolVar(&state.registry.showRegistry, paramShow, false, "show a list of items registered for monitoring")
	fsRegistry.StringVar(&state.registry.addItem, paramAdd, "", "add item to monitoring list")
	fsRegistry.BoolVar(&state.registry.showTypes, paramTypes, false, "show all eve item types")

	var err error
	if state.monitoring.db, err = connectToDatabase(); err != nil {
		panic(err)
	}
	state.monitoring.logger = ioutil.Discard

	return map[string]*flag.FlagSet{
		commandMonitoring: fsMonitoring,
		commandRegistry:   fsRegistry,
	}
}

// one-time execution of the tracking process
func monitorTick(state programState, registry map[int64]registryItem, checkContract bool, chSignal chan<- registrySignal) {
	var (
		conCh = make(chan contract, 10)
		errCh = make(chan error, 10)
	)
	go loadAllContracts(state.eve, state.monitoring.region, state.monitoring.logger, conCh, errCh)
	go func() {
		fmt.Fprintln(state.monitoring.logger, "started error reader thread")
		for err := range errCh {
			ifErrorPrint(err)
		}
		fmt.Fprintln(state.monitoring.logger, "closed error reader thread")
	}()
	fmt.Fprintln(state.monitoring.logger, "started contract reader thread")
	for contract := range conCh {
		if isPublicItemExchangeContract(contract) {
			isNew := isNewlyCreatedContractCheckDB(state.monitoring.db, contract)
			if isNew && checkContract {
				fmt.Fprintf(state.monitoring.logger, "got newly created: %d\n", contract.Id)
				if contract.Title != "" {
					fmt.Fprintln(state.monitoring.logger, contract.Title)
				}
				monCheckContract(contract, state.eve, registry, chSignal, state.monitoring.logger)
			}
		}
	}
	fmt.Fprintln(state.monitoring.logger, "closed contract reader thread")
}

func alerter(w io.Writer, registry map[int64]registryItem, chSignal <-chan registrySignal) {
	var lastTime = time.Now()
	for sig := range chSignal {
		fmt.Fprintln(w, "*********************************")
		fmt.Fprintln(w, sig.contract.Title)
		fmt.Fprintf(w, "Price: %0.3f M\n", sig.contract.Price/1000000)
		for _, s := range sig.items {
			fmt.Fprintln(w, "---------------------------------")
			if t, ok := registry[s.TypeId]; ok {
				fmt.Fprintf(w, "Item: %s\n", t.TypeName)
			}
			if s.Runs > 0 {
				fmt.Fprintf(w, "Quantity: %d\nRuns: %d\n", s.Quantity, s.Runs)
			} else {
				fmt.Fprintf(w, "Quantity: %d\nORIGINAL\n", s.Quantity)
			}
		}
		fmt.Fprintln(w, "*********************************")
		fmt.Fprintln(w, "")
		if lastTime.Add(time.Second * 20).Before(time.Now()) {
			ifErrorPrint(warning())
		}
	}
}

func startMonitoring(state programState) {
	fmt.Println("initialization...")
	registry := loadRegistry()
	if len(registry) == 0 {
		fmt.Fprintln(os.Stderr, "registry is empty")
		os.Exit(2)
	}
	// audio system initialization required
	ifErrorFatal(portaudio.Initialize())
	defer deferWithPrintError(portaudio.Terminate)

	var (
		checkContracts = false
		chSignal       = make(chan registrySignal, 10)
	)
	defer close(chSignal)
	go alerter(os.Stdout, registry, chSignal)
	for {
		<-time.After(time.Second * 5)
		monitorTick(state, registry, checkContracts, chSignal)
		if !checkContracts {
			checkContracts = true
			fmt.Fprintln(os.Stdout, "now we can start monitoring")
			ifErrorPrint(warning())
		}
	}
}

func registryOperations(state programState) {
	registry := loadRegistry()
	allTypes := state.eve.getItemTypes()
	if state.registry.showTypes {
		for _, t := range allTypes {
			fmt.Fprintf(state.output, "%d %s\n", t.typeId, t.typeName)
		}
	}
	if state.registry.addItem != "" {
		var (
			err     error
			doneStr string
		)
		registry, doneStr, err = addToRegistry(registry, allTypes, state.registry.addItem)
		ifErrorFatal(err)
		ifErrorFatal2(state.output.Write([]byte(doneStr)))
	}
	if state.registry.showRegistry {
		fmt.Fprintln(state.output, "monitoring list:")
		for _, i := range registry {
			fmt.Fprintf(state.output, "%s [%0.3f]\n", getItemName(allTypes, i.TypeId), i.Price)
		}
	}
}

func deferWithPrintError(fn func() error) {
	if err := fn(); err != nil {
		ifErrorPrint(err)
	}
}

func main() {
	var (
		state   programState
		command = "help"
	)
	if len(os.Args) > 1 {
		command = os.Args[1]
	}
	flagSets := state.init()
	if flagSet, ok := flagSets[command]; ok {
		if err := flagSet.Parse(os.Args[2:]); err != nil {
			panic(err)
		}
	} else {
		flagSets[commandMonitoring].Usage()
		flagSets[commandRegistry].Usage()
	}

	if command == commandMonitoring {
		if state.monitoring.verbose {
			state.monitoring.logger = os.Stdout
		}
		startMonitoring(state)
	}
	if command == commandRegistry {
		registryOperations(state)
	}
	os.Exit(1)
}
