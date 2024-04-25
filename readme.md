# PM (process manager)

## Installation
PM is available only for linux due to heavy usage of linux mechanisms. For now only installation way is to use `go install`:
```sh
go install github.com/rprtr258/pm@latest
```

### Systemd service
To enable running processes on system startup
- copy [`pm.service`](./pm.service) locally
- change `User` field to your own username
- move the file to `/etc/systemd/system/pm.service` and set root permissions on it:
```sh
sudo cp pm.service /etc/systemd/system/pm.service
sudo chmod 644 /etc/systemd/system/pm.service
sudo chown root:root /etc/systemd/system/pm.service
sudo systemctl daemon-reload
sudo systemctl enable pm
sudo systemctl start pm
```

After that, processes with `startup: true` config option will be started on system startup.

## Configuration
[jsonnet](https://jsonnet.org/) configuration language is used. It is also fully compatible with plain JSON, so you can write JSON instead.

See [example configuration file](./config.jsonnet). Other examples can be found in [tests](./tests) directory.

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
  RC -->|SIGCHLD| S
  RC -->|stop| S1
  RD -->|stop/process died| S1
  Stopped -->|start| RC
```

## Development

### Architecture
`pm` consists of two parts:
- cli client - requests server, launches/stops agent processes
- agent - monitors and restarts processes, handle signals and shutdowns

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

### Differences from [pm2](https://github.com/Unitech/pm2)
- `pm` is just a single binary, not dependent on `nodejs` and bunch of `js` scripts
- [jsonnet](https://jsonnet.org/) configuration language, back compatible with `JSON`, and allows to thoroughly configure processes, e.g. separate environments without requiring corresponding mechanism in `pm` (others configuration languages might be added in future such as `Procfile`, `HCL`, etc.)
- supports only `linux` now
- I can fix problems/add features as I need, independent of whether they work or not in `pm2`, because I don't know `js`
- fast and convenient (I hope so)
