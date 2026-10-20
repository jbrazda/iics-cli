package client

import (
	"sort"
	"strings"
)

// PrivilegeAction is the leading verb of an "<action>.<object>" privilege name.
type PrivilegeAction string

// Privilege actions recognized when grouping privileges by object.
const (
	ActionView       PrivilegeAction = "view"
	ActionCreate     PrivilegeAction = "create"
	ActionUpdate     PrivilegeAction = "update"
	ActionDelete     PrivilegeAction = "delete"
	ActionExecute    PrivilegeAction = "execute"
	ActionChangePerm PrivilegeAction = "changeperm"
)

// PrivilegeActions lists the recognized actions in display order.
var PrivilegeActions = []PrivilegeAction{
	ActionView, ActionCreate, ActionUpdate, ActionDelete, ActionExecute, ActionChangePerm,
}

// NoServiceGroup is the service name used for privileges with an empty service.
const NoServiceGroup = "(no service)"

// PrivilegeObject groups the action privileges that apply to one object type,
// e.g. "data.transfer.task" with its view/create/update/delete privileges.
type PrivilegeObject struct {
	Key     string
	Actions map[PrivilegeAction]Privilege
}

// PrivilegeService groups the privileges of one service. Privileges whose
// names do not follow "<action>.<object>" are listed in Other.
type PrivilegeService struct {
	Name    string
	Objects []PrivilegeObject
	Other   []Privilege
}

// Count returns the number of privileges in the service.
func (s PrivilegeService) Count() int {
	n := len(s.Other)
	for _, o := range s.Objects {
		n += len(o.Actions)
	}
	return n
}

// GroupPrivileges groups privileges by service, then by object and action.
// Services, objects and Other entries are sorted case-insensitively by name.
func GroupPrivileges(all []Privilege) []PrivilegeService {
	actions := make(map[string]PrivilegeAction, len(PrivilegeActions))
	for _, a := range PrivilegeActions {
		actions[string(a)] = a
	}

	type svcAcc struct {
		objects map[string]*PrivilegeObject
		other   []Privilege
	}
	services := make(map[string]*svcAcc)

	for _, p := range all {
		name := p.Service
		if name == "" {
			name = NoServiceGroup
		}
		acc, ok := services[name]
		if !ok {
			acc = &svcAcc{objects: make(map[string]*PrivilegeObject)}
			services[name] = acc
		}
		verb, object, found := strings.Cut(p.Name, ".")
		action, known := actions[verb]
		if !found || !known || object == "" {
			acc.other = append(acc.other, p)
			continue
		}
		obj, ok := acc.objects[object]
		if !ok {
			obj = &PrivilegeObject{Key: object, Actions: make(map[PrivilegeAction]Privilege)}
			acc.objects[object] = obj
		}
		obj.Actions[action] = p
	}

	result := make([]PrivilegeService, 0, len(services))
	for name, acc := range services {
		svc := PrivilegeService{Name: name, Other: acc.other}
		for _, o := range acc.objects {
			svc.Objects = append(svc.Objects, *o)
		}
		sort.Slice(svc.Objects, func(i, j int) bool {
			return lessFold(svc.Objects[i].Key, svc.Objects[j].Key)
		})
		sort.Slice(svc.Other, func(i, j int) bool {
			return lessFold(svc.Other[i].Name, svc.Other[j].Name)
		})
		result = append(result, svc)
	}
	sort.Slice(result, func(i, j int) bool {
		return lessFold(result[i].Name, result[j].Name)
	})
	return result
}

// DiffPrivileges returns the privilege names to add and remove to turn
// current into desired. Both results are sorted.
func DiffPrivileges(current, desired []string) (add, remove []string) {
	cur := make(map[string]bool, len(current))
	for _, n := range current {
		cur[n] = true
	}
	want := make(map[string]bool, len(desired))
	for _, n := range desired {
		want[n] = true
		if !cur[n] {
			add = append(add, n)
		}
	}
	for _, n := range current {
		if !want[n] {
			remove = append(remove, n)
		}
	}
	sort.Strings(add)
	sort.Strings(remove)
	return add, remove
}

func lessFold(a, b string) bool {
	la, lb := strings.ToLower(a), strings.ToLower(b)
	if la != lb {
		return la < lb
	}
	return a < b
}
