# IDWDetector

The dataset of IDWDetector is in "dataset".

The core code of IDWDetector is in "IDWDetector/hunter".

## Usage
Detecting IDW vulnerabilities from history transactions.
```
go run hunter/main.go
```

Detecting IDW vulnerabilities in DeFi apps by testing.
```
go run cmd/substate-cli/main.go test-hunter bzx 10000000 1 --substatedir <the path of substate>
```
