package logrotation

import "time"

type clock func() time.Time
