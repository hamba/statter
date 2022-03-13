package l2met_test

import (
	"os"
	"time"

	"github.com/hamba/logger/v2"
	"github.com/hamba/statter/v2"
	"github.com/hamba/statter/v2/reporter/l2met"
)

func ExampleNew() {
	log := logger.New(os.Stdout, logger.LogfmtFormat(), logger.Info)

	reporter := l2met.New(log, "my-prefix")

	statter.New(reporter, 10*time.Second)
}
