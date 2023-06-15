# PM (process manager)

## Installation
PM is available only for linux due to heavy usage of linux mechanisms. For now only installation way is to use `go install`:
```sh
go install github.com/rprtr258/pm@latest
```

## Configuration
[jsonnet](https://jsonnet.org/) configuration language is used. It is also fully compatible with plain JSON, so you can write JSON instead.

See [example configuration file](./config.jsonnet).

## Usage
Most fresh usage descriptions can be seen using `pm <command> --help`.

### Run process
```sh
# run process using command
pm run go run main.go

# run processes from config file
pm run --config config.jsonnet
```

### List processes
```sh
pm list
```

### Start processes that already has been added
```sh
pm start <id or name or tag...>
```

### Stop processes
```sh
pm stop <id or name or tag...>

# e.g. stop all added processes (all processes has tag `all` by default)
pm stop all
```

### Delete processes
When deleting process, they are first stopped, then removed from `pm`.
```sh
pm delete <id or name or tag...>

# e.g. delete all processes
pm delete all
```


## Process state diagram
```mermaid
flowchart TB
  C[Created]
  RC[Running/Child]
  RD[Running/Detached]
  subgraph Stopped
    S
    S1
  end
  subgraph Running
    direction TB
    RC -->|daemon restart| RD
  end
  S["Stopped(ExitCode)"]
  S1["Stopped(-1)"]
  C -->|start| RC
  RC -->|stop/SIGCHLD| S
  RD -->|stop/process died| S1
  Stopped -->|start| RC
```

## Development

### PM directory structure
`pm` uses directory `$HOME/.pm` to store data. Layout is following:

```sh
$HOME/.pm/
├── config.json # pm config file
├── rpc.sock    # socket used to rpc daemon
├── pm.log      # daemon logs
├── pm.pid      # daemon pid
├── db/ # database tables
│   └── procs.json # processes table
└── logs/ # processes logs
    ├── <ID>.stdout # stdout of process with id ID
    └── <ID>.stderr # stderr of process with id ID
```
