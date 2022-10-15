package passwd

import (
	"bufio"
	"os"
	"strings"

	"github.com/willdonnelly/passwd"
)

type GroupEntry struct {
	password string
	id       string
	members  []string
}

func GetUsers() (map[string]passwd.Entry, error) {
	// TODO: abstract /etc/{group,passwd} parsers and use it here
	return passwd.Parse()
}

func GetGroups() (map[string]GroupEntry, error) {
	file, err := os.Open("/etc/group")
	if err != nil {
		return nil, err
	}
	defer file.Close() // TODO: handle error

	scanner := bufio.NewScanner(file)
	// name -> group
	res := make(map[string]GroupEntry)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		fields := strings.Split(line, ":")
		name, password, id, members := fields[0], fields[1], fields[2], strings.Split(fields[3], ",")
		// TODO: by name or by id?
		res[name] = GroupEntry{
			password: password,
			id:       id,
			members:  members,
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return res, nil
}

// module.exports = {
//   getUsers,
//   getGroups
// }
