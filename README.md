# EVE contract monitor

This utility was created to help you keep track of blueprints in `the Forge` region.  
When the contracts for the exchange of items are created, they will immediately be checked for the price, if the price suits you, you will receive a signal.  

## How to use it

### List typeIDs

To register an item, you need to know its ID. A list of all IDs can be obtained using this command:  
```shell script
jitaScan registry --types
```

To filter the result by the required parameters (name), use the grep utility:  
```shell script
jitaScan registry --types|grep Gila
```

### Register contracts

To add an item to the watchlist, enter this command:  
```shell script
jitaScan registry --add "17931 45"
```
Here the text in quotation marks means `typeID` and `amount in millions of ISK` separated by a space.  

To see what types are already registered for observation use this command:  
```shell script
jitaScan registry --show
```

### Start monitoring

To start monitoring use this command:  
```shell script
jitaScan monitoring
```

## What next?

The utility runs and performs the following actions:  

  1. Scans all contracts* in the "Forge" region and remembers their ID
  2. Waits 5 seconds
  3. Scans all contracts* in the "Forge" region and finds among them those with new ID
  4. Saves these new IDs
  5. For each of these IDs loads a list of items
  6. For each item that is tied to the contract and at the same time registered for monitoring, sums up the price that you specified
  7. If the real amount of the contract is less than or equal to the amount calculated in the previous step, a sound signal is played and the data about the contract goes to the screen
  8. Everything repeats from step 2
  
*only those contracts are considered for which type is `items exchange`