# MSR Paper: "TypeScript's Evolution: An Analysis of Feature Adoption Over Time"

## Replication

We include the extracted feature and commit data for our data analysis scripts in the repository. You can reproduce our graphs and statistics using just the included data.

### Requirements

- Golang version go1.18.4
- Python 3.10
- Nodejs v18.9.0

### Collect Repository List

Put a valid GitHub token in `tools/ghToken.txt`

```
go run cmd/getrepos/getrepos.go
```

### Clean Repository List

```
go run cmd/processrepos/processrepos.go
```

### Download Repositories

```
go run cmd/download/download.go
```

### Run Data Collection

```
go run cmd/collect/collect.go
```

### Run Data Analysis

```
python tools/analyse.py
```

### Generate Graphs

```
python tools/generate_graphs.py
```
