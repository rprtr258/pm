# Process Manager

### Technical debt

- [ ] use grpc status codes for server response errors  
- [ ] serialize/deserialize protobuffers into database  
- [ ] manage ids ourself, leveraging minimal not used id for new proc  
- [ ] draw processes states/transitions diagrams  

### Bugfixes

- [ ] delete cmd must also delete log files  
- [ ] fix not showing error on failed start tests/main.go second time  
- [ ] fix someone holding $home/pm.pid file on daemon restart  
- [ ] fix writing,creating pid file for the first time  

### Features

- [ ] administrative tasks for processes: pull repo, seed db, etc.  
- [ ] config file with process definitions  
- [ ] run specific processes from config  
- [ ] pause/return processes from config  
- [ ] specify cwd in config  
- [ ] jsonnet config  
- [ ] version commands  
- [ ] daemon restart commands  
- [ ] implement different list formats: table, short list, json, go format template  
- [ ] gen name if not provided  
- [ ] pm start recognizing cmd&args vs ids/names/tags of processes in pm list to run  
- [ ] add "smart filtering" to delete and stop cmds  
- [ ] -i/... flag to confirm which procs will be stopped  
- [ ] bash autocomplete  
- [ ] provide envs from dotenv/...  
- [ ] start only processes with changed config  
- [ ] watch: restart process on file change [fsnotify](https://github.com/fsnotify/fsnotify) [modd](https://github.com/cortesi/modd) [entr](https://github.com/eradman/entr) [reflex](https://github.com/cespare/reflex) [air](https://github.com/cosmtrek/air)  
- [ ] run and attach (e.g. interactively), stop on Ctrl-C  
- [ ] show logs of several proccesses simultanuously  
- [ ] logrotaion: [lumberjack](https://github.com/natefinch/lumberjack) [pm2-logrotate](https://github.com/keymetrics/pm2-logrotate)  
- [ ] dashboard: [pm-web](https://github.com/VividCortex/pm-web) [pm2-server-monit](https://github.com/keymetrics/pm2-server-monit) [pm2-dev](https://github.com/Unitech/pm2-dev)  
- [ ] make executable + arguments run options be available  
- [ ] run daemon if not running before executing (almost) any command  
- [ ] [self autoupdate](https://developers.redhat.com/articles/2022/11/14/3-ways-embed-commit-hash-go-programs)  

### Doing

- [ ] try [badger-db](https://github.com/dgraph-io/badger) [get-started](https://dgraph.io/docs/badger/get-started/)  

### Done


