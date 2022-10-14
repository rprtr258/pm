package pm

import "os"

func init() {
	os.Setenv("PM2_PROGRAMMATIC", "true")
}
