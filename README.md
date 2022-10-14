# pm

PM2 is a production process manager for Node.js applications with a built-in load balancer. It allows you to keep applications alive forever, to reload them without downtime and to facilitate common system admin tasks.

Starting an application in production mode is as easy as:

```bash
pm2 start app.js
```

PM2 is constantly assailed by [more than 1800 tests](https://app.travis-ci.com/github/Unitech/pm2).

Official website: [https://pm2.keymetrics.io/](https://pm2.keymetrics.io/)

Works on Linux (stable) & macOS (stable) & Windows (stable). All Node.js versions are supported starting Node.js 12.X.


### Installing PM2

With NPM:

```bash
npm install pm2 -g
```

You can install Node.js easily with [NVM](https://github.com/nvm-sh/nvm#installing-and-updating) or [ASDF](https://blog.natterstefan.me/how-to-use-multiple-node-version-with-asdf).

### Start an application

You can start any application (Node.js, Python, Ruby, binaries in $PATH...) like that:

```bash
pm2 start app.js
```

Your app is now daemonized, monitored and kept alive forever.

### Managing Applications

Once applications are started you can manage them easily:

![Process listing](https://github.com/Unitech/pm2/raw/master/pres/pm2-ls-v2.png)

To list all running applications:

```bash
pm2 list
```

Managing apps is straightforward:

```bash
pm2 stop     <app_name|namespace|id|'all'|json_conf>
pm2 restart  <app_name|namespace|id|'all'|json_conf>
pm2 delete   <app_name|namespace|id|'all'|json_conf>
```

To have more details on a specific application:

```bash
pm2 describe <id|app_name>
```

To monitor logs, custom metrics, application information:

```bash
pm2 monit
```

[More about Process Management](https://pm2.keymetrics.io/docs/usage/process-management/)

### Cluster Mode: Node.js Load Balancing & Zero Downtime Reload

The Cluster mode is a special mode when starting a Node.js application, it starts multiple processes and load-balance HTTP/TCP/UDP queries between them. This increase overall performance (by a factor of x10 on 16 cores machines) and reliability (faster socket re-balancing in case of unhandled errors).

![Framework supported](https://raw.githubusercontent.com/Unitech/PM2/master/pres/cluster.png)

Starting a Node.js application in cluster mode that will leverage all CPUs available:

```bash
pm2 start api.js -i <processes>
```

`<processes>` can be `'max'`, `-1` (all cpu minus 1) or a specified number of instances to start.

**Zero Downtime Reload**

Hot Reload allows to update an application without any downtime:

```bash
pm2 reload all
```

[More informations about how PM2 make clustering easy](https://pm2.keymetrics.io/docs/usage/cluster-mode/)

### Container Support

With the drop-in replacement command for `node`, called `pm2-runtime`, run your Node.js application in a hardened production environment.
Using it is seamless:

```
RUN npm install pm2 -g
CMD [ "pm2-runtime", "npm", "--", "start" ]
```

[Read More about the dedicated integration](https://pm2.keymetrics.io/docs/usage/docker-pm2-nodejs/)

### Host monitoring speedbar

PM2 allows to monitor your host/server vitals with a monitoring speedbar.

To enable host monitoring:

```bash
pm2 set pm2:sysmonit true
pm2 update
```

![Framework supported](https://raw.githubusercontent.com/Unitech/PM2/master/pres/vitals.png)

### Terminal Based Monitoring

![Monit](https://github.com/Unitech/pm2/raw/master/pres/pm2-monit.png)

Monitor all processes launched straight from the command line:

```bash
pm2 monit
```

### Log Management

To consult logs just type the command:

```bash
pm2 logs
```

Standard, Raw, JSON and formated output are available.

Examples:

```bash
pm2 logs APP-NAME       # Display APP-NAME logs
pm2 logs --json         # JSON output
pm2 logs --format       # Formated output

pm2 flush               # Flush all logs
pm2 reloadLogs          # Reload all logs
```

To enable log rotation install the following module

```bash
pm2 install pm2-logrotate
```

[More about log management](https://pm2.keymetrics.io/docs/usage/log-management/)

### Startup Scripts Generation

PM2 can generate and configure a Startup Script to keep PM2 and your processes alive at every server restart.

Init Systems Supported: **systemd**, **upstart**, **launchd**, **rc.d**

```bash
# Generate Startup Script
pm2 startup

# Freeze your process list across server restart
pm2 save

# Remove Startup Script
pm2 unstartup
```

[More about Startup Scripts Generation](https://pm2.keymetrics.io/docs/usage/startup/)

### Updating PM2

```bash
# Install latest PM2 version
npm install pm2@latest -g
# Save process list, exit old PM2 & restore all processes
pm2 update
```

*PM2 updates are seamless*

## PM2+ Monitoring

If you manage your apps with PM2, PM2+ makes it easy to monitor and manage apps across servers.

![https://app.pm2.io/](https://pm2.io/img/app-overview.png)

Feel free to try it:

[Discover the monitoring dashboard for PM2](https://app.pm2.io/)

Thanks in advance and we hope that you like PM2!

# Contributing

## Cloning PM2 development

```bash
git clone https://github.com/Unitech/pm2.git
cd pm2
git checkout development
npm install
```

I recommend having a pm2 alias pointing to the development version to make it easier to use pm2 development:

```
cd pm2/
echo "alias pm2='`pwd`/bin/pm2'" >> ~/.bashrc
```

You are now able to use pm2 in dev mode:

```
pm2 update
pm2 ls
```

## Project structure

```
.
├── bin      // pm2, pmd, pm2-dev, pm2-docker are there
├── examples // examples files
├── lib      // source files
├── pres     // presentation files
├── test     // test files
└── types    // TypeScript definition files
```

## Modifying the Daemon

When you modify the Daemon (lib/Daemon.js, lib/God.js, lib/God/*, lib/Watcher.js), you must restart the pm2 Daemon by doing:

```
pm2 update
```

## Commit rules

### Commit message

A good commit message should describe what changed and why.

It should :
  * contain a short description of the change (preferably 50 characters or less)
  * be entirely in lowercase with the exception of proper nouns, acronyms, and the words that refer to code, like function/variable names
  * be prefixed with one of the following word
    * fix : bug fix
    * hotfix : urgent bug fix
    * feat : new or updated feature
    * docs : documentation updates
    * BREAKING : if commit is a breaking change
    * refactor : code refactoring (no functional change)
    * perf : performance improvement
    * style : UX and display updates
    * test : tests and CI updates
    * chore : updates on build, tools, configuration ...
    * Merge branch : when merging branch
    * Merge pull request : when merging PR

## Tests

There are two tests type. Programmatic and Behavioral.
The main test command is `npm test`

### Programmatic

Programmatic tests are runned by doing

```
bash test/pm2_programmatic_tests.sh
```

This test files are located in test/programmatic/*

### Behavioral

Behavioral tests are runned by doing:

```
bash test/e2e.sh
```

This test files are located in test/e2e/*

## File of interest

- `$HOME/.pm2` contain all PM2 related files
- `$HOME/.pm2/logs` contain all applications logs
- `$HOME/.pm2/pids` contain all applications pids
- `$HOME/.pm2/pm2.log` PM2 logs
- `$HOME/.pm2/pm2.pid` PM2 pid
- `$HOME/.pm2/rpc.sock` Socket file for remote commands
- `$HOME/.pm2/pub.sock` Socket file for publishable events

## Generate changelog

### requirements

```
npm install git-changelog -g
```

### usage

Edit .changelogrc
Change "version_name" to the next version to release (example 1.1.2).
Change "tag" to the latest existing tag (example 1.1.1).

Run the following command into pm2 directory
```
git-changelog
```

It will generate currentTagChangelog.md file.
Just copy/paste the result into changelog.md

# Contributing to PM2

## Pull-Requests

1. Fork pm2
2. Create a different branch to do your fixes/improvements if it's core-related.
3. Please add unit tests! There are lots of tests take examples from there!
4. Try to be as clear as possible in your commits
5. Pull request on the **development branch** from your fork

We 'd like to keep our master branch as clean as possible, please avoid PRs on master, thanks!

## Fire an issue

When you got an issue by using pm2, you will fire an issue on [github](https://github.com/Unitech/pm2). We'll be glad to help or to fix it but the more data you give the most fast it would be resolved.
Please try following these rules it will make the task easier for you and for us:

#### 1. Search through issues if it hasn't been resolved yet
#### 2. Make sure that you provide following informations:
  - pm2 version `pm2 --version`
  - nodejs version `node --version`
  - operating system
  - `head -n 50 ~/.pm2/pm2.log` output

#### 3. Provide details about your issue:
  - What are the steps that brought me to the issue?
  - How may I reproduce this? (this isn't easy in some cases)
  - Are you using a cluster module? Are you trying to catch SIGTERM signals? With `code` if possible.

#### 4. Think global
If your issue is too specific we might not be able to help and stackoverflow might be a better place to seak for an answer

#### 5. Be clear and format issues with [markdown](http://daringfireball.net/projects/markdown/)
Note that we might understand english, german and french but english is prefered.

#### 6. Use debugging functions:

```DEBUG=pm2:* PM2_DEBUG=true ./bin/pm2 --no-daemon start my-buggy-thing.js```

If your issue is flagged as `need data` be sure that there won't be any upgrade unless we can have enough data to reproduce.
