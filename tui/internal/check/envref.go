package check

import (
	"context"
	"regexp"
)

// TextSource is a text scanned for variable references; Subject names where it came from.
type TextSource struct{ Subject, Text string }

// EnvRefs reports each reference matched by pattern (group 1 = name, group 2 = default); values are never read into results.
func EnvRefs(id string, pattern *regexp.Regexp, extract func(env Env) ([]TextSource, error)) Check {
	return Check{ID: id, Run: func(_ context.Context, env Env) []Result {
		sources, err := extract(env)
		if err != nil {
			return []Result{{CheckID: id, Subject: id, Status: StatusUnknown, Message: err.Error()}}
		}
		var results []Result
		for _, src := range sources {
			for _, m := range pattern.FindAllStringSubmatch(src.Text, -1) {
				name, hasDefault := m[1], m[2] != ""
				r := Result{CheckID: id, Subject: src.Subject, Status: StatusOK, Message: name + " is set"}
				if _, set := env.LookupEnv(name); !set {
					if hasDefault {
						r.Message = name + " is unset, default applies"
					} else {
						r.Status, r.Message = StatusWarning, name+" is not set"
						r.Suggestion = "set " + name + " in the environment, or use ${" + name + ":-default}"
					}
				}
				results = append(results, r)
			}
		}
		return results
	}}
}
