# netps

## Run tests
### Run fuzz tests
```
go test -fuzz=FuzzCoordinatorStateMachine -run=^$
GOMAXPROCS=8 go test -race -fuzz=FuzzCoordinatorConcurrentStateMachine -run=^$
go test -fuzz=FuzzFieldWouldChange -run=^$
```
