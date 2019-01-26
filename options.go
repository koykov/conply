package conply

import (
	"fmt"
	"strings"
)

type Options map[string]interface{}

func (o *Options) PrettyPrint() string {
	var res []string
	for k, v := range *o {
		res = append(res, fmt.Sprintf(" * %s: %v", k, v))
	}
	return strings.Join(res, "\n")
}
