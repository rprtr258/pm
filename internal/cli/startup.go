package cli

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"

	"github.com/spf13/cobra"

	"github.com/rprtr258/pm/internal/errors"
)

var _cmdStartupSystemd = func() *cobra.Command {
	var username string
	cmd := &cobra.Command{
		Use:    "systemd",
		Short:  "install systemd service",
		Args:   cobra.NoArgs,
		Hidden: true,
		RunE: func(c *cobra.Command, _ []string) error {
			err := os.WriteFile("/etc/systemd/system/pm.service", []byte(`[Unit]
Description=PM process manager
After=network.target

[Service]
Type=forking
User=`+username+`
Group=`+username+`
ExecStart=/usr/bin/pm `+_cmdRunStartup.Use+`

[Install]
WantedBy=multi-user.target
`), 0o640)
			if err != nil {
				return errors.Wrap(err, "write pm.service")
			}

			for _, cmd := range [][]string{
				// set permission of service file to be readable and writable by owner, and readable by others
				{"chmod", "644", "/etc/systemd/system/pm.service"},
				// change owner and group of service file to root, ensuring that it is managed by system administrator
				{"chown", "root:root", "/etc/systemd/system/pm.service"},
				// reload systemd manager configuration, scanning for new or changed units
				{"systemctl", "daemon-reload"},
				// enables service to start at boot time
				{"systemctl", "enable", "pm"},
				// starts service immediately
				{"systemctl", "start", "pm"},
			} {
				cmd := exec.CommandContext(c.Context(), cmd[0], cmd[1:]...) //nolint:gosec // hardcoded commands
				if err := cmd.Run(); err != nil {
					return errors.Wrapf(err, "run command %q", cmd.String())
				}
			}

			return nil
		},
	}
	cmd.Flags().StringVarP(&username, "username", "u", "", "username to run pm as")
	return cmd
}()

var _cmdStartup = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "startup",
		Short: "run startup processes",
		Args:  cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			user, err := user.Current()
			if err != nil {
				return errors.Wrap(err, "get current user")
			}

			bin, err := os.Executable()
			if err != nil {
				return errors.Wrap(err, "get executable path")
			}

			fmt.Fprintln(os.Stderr, "# To install systemd startup service, copy/paste the following command:")
			fmt.Fprintf(os.Stderr, "sudo %s startup systemd -u %s\n", bin, user.Name)

			return nil
		},
	}
	cmd.AddCommand(_cmdStartupSystemd)
	return cmd
}()
