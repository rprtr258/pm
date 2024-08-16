# PM (process manager)
## Installation
PM is available only for linux due to heavy usage of linux mechanisms. Go to the [releases](https://github.com/rprtr258/pm/releases/latest) page to download the latest binary.

```sh
# download binary
wget https://github.com/rprtr258/pm/releases/latest/download/pm_linux_amd64
# make binary executable
chmod +x pm_linux_amd64
# move binary to $PATH, here just local
mv pm_linux_amd64 pm
```

### Systemd service
To enable running processes on system startup:
- Copy [pm.service](./pm.service) file locally. This is the systemd service file that tells systemd how to manage your application.
- Change `User` field to your own username. This specifies under which user account the service will run, which affects permissions and environment.
- Change `ExecStart` to use `pm` binary installed. This is the command that systemd will execute to start your service.
- Move the file to `/etc/systemd/system/pm.service` and set root permissions on it:
```sh
# copy service file to system's directory for systemd services
sudo cp pm.service /etc/systemd/system/pm.service
# set permission of service file to be readable and writable by owner, and readable by others
sudo chmod 644 /etc/systemd/system/pm.service
# change owner and group of service file to root, ensuring that it is managed by system administrator
sudo chown root:root /etc/systemd/system/pm.service
# reload systemd manager configuration, scanning for new or changed units
sudo systemctl daemon-reload
# enables service to start at boot time
sudo systemctl enable pm
# starts service immediately
sudo systemctl start pm
# soft link /usr/bin/pm binary to whenever it is installed
sudo ln -s ~/go/bin/pm /usr/bin/pm
```
After these commands, processes with `startup: true` config option will be started on system startup.

## Configuration
[jsonnet](https://jsonnet.org/) configuration language is used. It is also fully compatible with plain JSON, so you can write JSON instead.

See [example configuration file](./config.jsonnet). Other examples can be found in [tests](./tests) directory.
## Usage
### Run process
### List processes
### Start already added processes
### Stop processes
### Delete processes
## Process state diagram
## Development
### Architecture
### PM directory structure
### Differences from pm2
### Release