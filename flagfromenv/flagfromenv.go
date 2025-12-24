package flagfromenv

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

// Lists flags, and sets them from environment variables.
// my.flag.name-with-dashes would by set by environmentVariablePrefix+_MY__FLAG__NAME_WITH_DASHES
// '.' -> two underscores
// '-' -> single underscore
// flag.Parse() is expected to be called first.
// Environment variables will overwrite flags even if explicitly set.
func Parse(environmentVariablePrefix string) (err error) {
	flag.VisitAll(func(f *flag.Flag) {
		replacer := strings.NewReplacer(
			".", "__",
			"-", "_",
		)
		if val, ok := os.LookupEnv(environmentVariablePrefix + "_" + replacer.Replace(strings.ToUpper(f.Name))); ok {
			err = flag.Set(f.Name, val)
			if err != nil {
				err = fmt.Errorf("failed to set flag: %w", err)
				return
			}
		}
	})
	return err
}
