package business

import (
	"strings"

	"github.com/pitabwire/util"
)

func prefixedID(prefix string) string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return util.IDString()
	}

	return prefix + util.IDString()
}
