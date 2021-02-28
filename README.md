# Quatermain

An explorer tool for generate a dynamic sitemap

## Requirements
- `go v1.16`

## Install

`git clone git@github.com:CasvalDOT/quatermain.git`

`go mod vendor`

`go build`

`cp quatermain /usr/bin/`

## Usage

quatermain <url>/

*Note! The initial URL must be finish with /*

```
quatermain https://mydomain.com/
```

You can change for now the following scripts parameters using the flags:
- **maxConnections** with the flag **-mc** is the max of goroutines allowed to run in the same time. Default value is **120**.
- **heartBeatInterval** with the flag **-hb** is the check interval in seconds until script finish to scan. Default value is 10 seconds.

```
quatermain -mc 200 -hb 120 https://mydomain.com/
```

The command provided start scan mydomain.com with a maximum pool of connections of 200 and stop when the script is inactive after 120 seconds
