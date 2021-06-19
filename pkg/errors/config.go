package errors

import (
	"fmt"
	"strings"
)

// Config returns an error specifically tailored for reporting errors with configuration
// values. You can check for these errors by calling errors.Is(err, KindConfigError)
// (from the github.com/gomods/athens/pkg/errors package).
//
// Generally these kinds of errors should make Athens crash because it was configured
// improperly
func Config(op Op, field, helpText, url string) error {
	slc := []string{
		fmt.Sprintf("There was a configuration error with %s. %s", field, helpText),
	}
	if url != "" {
		slc = append(slc, fmt.Sprintf("Please see %s for more information.", url))
	}
	return E(op, KindConfigError, strings.Join(slc, "\n\t"))
}
