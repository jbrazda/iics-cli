package tui

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/jbrazda/iics-cli/internal/client"
)

// AgentConfigEditorOptions configures EditAgentConfigs.
type AgentConfigEditorOptions struct {
	// Title is shown above each screen, e.g. "Agent: CAI-A03".
	Title string
}

// agentConfigEditor holds the properties and the pending edits by key.
type agentConfigEditor struct {
	props   []client.AgentConfigProperty
	pending map[string]client.AgentConfigChange
	title   string
}

const (
	choiceBack    = "\x00back"
	choiceAdd     = "\x00add"
	choiceText    = "\x00text"
	choiceReset   = "\x00reset"
	choiceUndo    = "\x00undo"
	choiceEdit    = "\x00edit"
	choiceRemove  = "\x00remove"
	choiceApply   = "\x00apply"
	previewWidth  = 44
	maxListHeight = 22
)

// EditAgentConfigs runs the interactive agent property editor and returns
// the changes to apply, in property order. Esc goes back one screen; on the
// service screen it cancels with ErrCanceled.
func EditAgentConfigs(props []client.AgentConfigProperty, opts AgentConfigEditorOptions) ([]client.AgentConfigChange, error) {
	e := &agentConfigEditor{props: props, pending: map[string]client.AgentConfigChange{}, title: opts.Title}
	lastService := ""
	for {
		choice := lastService
		options := e.serviceOptions()
		sel := huh.NewSelect[string]().
			Title(e.title).
			Description(fmt.Sprintf("Select a service (/ to filter). %d pending change(s).", len(e.pending))).
			Options(options...).
			Height(min(len(options)+2, maxListHeight)).
			Value(&choice)
		if err := runForm(huh.NewForm(huh.NewGroup(sel))); err != nil {
			return nil, err
		}
		switch choice {
		case choiceCancel:
			return nil, ErrCanceled
		case choiceReview:
			ok, err := e.review()
			if err != nil {
				return nil, err
			}
			if ok {
				return e.changes(), nil
			}
			continue
		}
		lastService = choice
		if err := e.editService(choice); err != nil {
			return nil, err
		}
	}
}

// back runs a form; Esc returns false instead of an error, so inner screens
// go back one level.
func back(f *huh.Form) (bool, error) {
	err := runForm(f)
	if errors.Is(err, ErrCanceled) {
		return false, nil
	}
	return err == nil, err
}

// value returns the effective value of property i with pending edits.
func (e *agentConfigEditor) value(i int) (string, bool) {
	p := e.props[i]
	ch, ok := e.pending[p.Key()]
	switch {
	case !ok:
		return p.Value, p.Overridden
	case ch.Reset:
		return p.DefaultValue, false
	}
	return ch.Value, true
}

// record stores the pending edit for property i, or drops it when the
// property ends up as it is now.
func (e *agentConfigEditor) record(i int, ch client.AgentConfigChange) {
	p := e.props[i]
	ch.Property = p.Key()
	if ch.Reset && !p.Overridden && p.Value == p.DefaultValue || !ch.Reset && ch.Value == p.Value {
		delete(e.pending, p.Key())
		return
	}
	e.pending[p.Key()] = ch
}

func (e *agentConfigEditor) changes() []client.AgentConfigChange {
	var out []client.AgentConfigChange
	for _, p := range e.props {
		if ch, ok := e.pending[p.Key()]; ok {
			out = append(out, ch)
		}
	}
	return out
}

func (e *agentConfigEditor) serviceOptions() []huh.Option[string] {
	type stats struct {
		name                 string
		total, over, changed int
	}
	var order []string
	byService := map[string]*stats{}
	width := 0
	for i, p := range e.props {
		s := byService[p.Service]
		if s == nil {
			s = &stats{name: p.ServiceName}
			if s.name == "" {
				s.name = p.Service
			}
			byService[p.Service] = s
			order = append(order, p.Service)
			width = max(width, len(s.name))
		}
		s.total++
		if _, over := e.value(i); over {
			s.over++
		}
		if _, ok := e.pending[p.Key()]; ok {
			s.changed++
		}
	}
	opts := make([]huh.Option[string], 0, len(order)+2)
	for _, svc := range order {
		s := byService[svc]
		label := fmt.Sprintf("%-*s  %4d properties  %3d overridden", width, s.name, s.total, s.over)
		if s.changed > 0 {
			label += fmt.Sprintf("  %d changed", s.changed)
		}
		opts = append(opts, huh.NewOption(label, svc))
	}
	return append(opts,
		huh.NewOption("> Review and apply", choiceReview),
		huh.NewOption("x Cancel", choiceCancel))
}

// editService lists the properties of one service until the user goes back.
func (e *agentConfigEditor) editService(service string) error {
	last := ""
	for {
		var idx []int
		typeW, nameW := 0, 0
		for i, p := range e.props {
			if p.Service == service {
				idx = append(idx, i)
				typeW, nameW = max(typeW, len(p.Type)), max(nameW, len(p.Name))
			}
		}
		if len(idx) == 0 {
			return nil
		}
		nameW = min(nameW, 32)
		opts := []huh.Option[string]{huh.NewOption("< Back to services", choiceBack)}
		for _, i := range idx {
			p := e.props[i]
			v, over := e.value(i)
			mark := " "
			if over {
				mark = "*"
			}
			if _, ok := e.pending[p.Key()]; ok {
				mark = "✎"
			}
			label := fmt.Sprintf("%s %-*s  %-*s  %s", mark, typeW, p.Type, nameW, truncate(p.Name, nameW), previewValue(v, p.Sensitive, previewWidth))
			opts = append(opts, huh.NewOption(label, strconv.Itoa(i)))
		}
		choice := last
		name := e.props[idx[0]].ServiceName
		sel := huh.NewSelect[string]().
			Title(e.title + " - " + name).
			Description("Select a property to edit (/ to filter). * overridden, ✎ changed.").
			Options(opts...).
			Height(min(len(opts)+2, maxListHeight)).
			Value(&choice)
		ok, err := back(huh.NewForm(huh.NewGroup(sel)))
		if err != nil || !ok || choice == choiceBack {
			return err
		}
		last = choice
		i, _ := strconv.Atoi(choice)
		if err := e.editProperty(i); err != nil {
			return err
		}
	}
}

// editProperty edits one property: lists of maps in the entry editor, other
// values in a text field.
func (e *agentConfigEditor) editProperty(i int) error {
	p := e.props[i]
	for {
		v, over := e.value(i)
		list, isList := parseEntryList(v)
		if !isList && strings.TrimSpace(v) == "" {
			// An empty list property: use the default's entries as the template.
			if def, ok := parseEntryList(p.DefaultValue); ok {
				list, isList = def, true
				for list.Len() > 0 {
					list.Remove(0)
				}
			}
		}
		state := "default"
		if over {
			state = "overridden"
		}
		if _, ok := e.pending[p.Key()]; ok {
			state += ", changed"
		}
		desc := fmt.Sprintf("Type %s · %s\nDefault: %s", p.Type, state, previewValue(p.DefaultValue, p.Sensitive, 60))

		var opts []huh.Option[string]
		if isList {
			for k := 0; k < list.Len(); k++ {
				opts = append(opts, huh.NewOption(fmt.Sprintf("%2d. %s", k+1, truncate(list.Label(k), 70)), strconv.Itoa(k)))
			}
			opts = append(opts,
				huh.NewOption("+ Add entry", choiceAdd),
				huh.NewOption("Edit as text", choiceText))
		} else {
			desc += "\nValue:   " + previewValue(v, p.Sensitive, 60)
			opts = append(opts, huh.NewOption("Edit value", choiceEdit))
		}
		if over {
			opts = append(opts, huh.NewOption("Reset to default", choiceReset))
		}
		if _, ok := e.pending[p.Key()]; ok {
			opts = append(opts, huh.NewOption("Undo changes to this property", choiceUndo))
		}
		opts = append(opts, huh.NewOption("< Back", choiceBack))

		choice := opts[0].Value
		sel := huh.NewSelect[string]().
			Title(fmt.Sprintf("%s › %s › %s", p.ServiceName, p.Type, p.Name)).
			Description(desc).
			Options(opts...).
			Height(min(len(opts)+2, maxListHeight)).
			Value(&choice)
		ok, err := back(huh.NewForm(huh.NewGroup(sel)))
		if err != nil || !ok || choice == choiceBack {
			return err
		}

		switch choice {
		case choiceReset:
			e.record(i, client.AgentConfigChange{Reset: true})
		case choiceUndo:
			delete(e.pending, p.Key())
		case choiceEdit:
			nv, changed, err := editScalar(p, v)
			if err != nil {
				return err
			}
			if changed {
				e.record(i, client.AgentConfigChange{Value: nv})
			}
		case choiceText:
			nv, changed, err := editText(p, v)
			if err != nil {
				return err
			}
			if changed {
				e.record(i, client.AgentConfigChange{Value: nv})
			}
		default: // add or an entry index
			if err := e.editEntry(i, list, choice); err != nil {
				return err
			}
		}
	}
}

// editEntry adds, changes or removes one entry of a list property.
func (e *agentConfigEditor) editEntry(i int, list *entryList, choice string) error {
	p := e.props[i]
	var keys, values []string
	k := -1
	if choice == choiceAdd {
		keys = list.Keys()
		values = make([]string, len(keys))
	} else {
		k, _ = strconv.Atoi(choice)
		for _, f := range list.Fields(k) {
			keys, values = append(keys, f[0]), append(values, f[1])
		}
		action := choiceEdit
		sel := huh.NewSelect[string]().
			Title(fmt.Sprintf("%s › %s › %s › entry %d", p.ServiceName, p.Type, p.Name, k+1)).
			Description(list.Label(k)).
			Options(
				huh.NewOption("Edit entry", choiceEdit),
				huh.NewOption("Remove entry", choiceRemove),
				huh.NewOption("< Back", choiceBack)).
			Value(&action)
		ok, err := back(huh.NewForm(huh.NewGroup(sel)))
		if err != nil || !ok || action == choiceBack {
			return err
		}
		if action == choiceRemove {
			list.Remove(k)
			return e.recordList(i, list)
		}
	}

	fields := make([]huh.Field, len(keys))
	for n := range keys {
		fields[n] = huh.NewInput().Title(keys[n]).Value(&values[n])
	}
	title := fmt.Sprintf("%s › %s › %s › new entry", p.ServiceName, p.Type, p.Name)
	if k >= 0 {
		title = fmt.Sprintf("%s › %s › %s › entry %d", p.ServiceName, p.Type, p.Name, k+1)
	}
	form := huh.NewForm(huh.NewGroup(fields...).Title(title).Description("Values are written as typed. Enter on the last field saves; Esc goes back."))
	ok, err := back(form)
	if err != nil || !ok {
		return err
	}
	if k >= 0 {
		list.Set(k, values)
	} else {
		list.Add(keys, values)
	}
	return e.recordList(i, list)
}

func (e *agentConfigEditor) recordList(i int, list *entryList) error {
	v, err := list.String()
	if err != nil {
		return err
	}
	e.record(i, client.AgentConfigChange{Value: v})
	return nil
}

// editScalar edits a one-line value. Values in YAML single quotes are edited
// without them and quoted again on save.
func editScalar(p client.AgentConfigProperty, v string) (string, bool, error) {
	plain, quoted := unquoteValue(v)
	if v == "" {
		_, quoted = unquoteValue(p.DefaultValue)
	}
	desc := "Default: " + previewValue(p.DefaultValue, p.Sensitive, 60)
	if quoted {
		desc += "\nThe value is stored in single quotes; they are added for you."
	}
	if strings.Contains(v, "\n") {
		return editText(p, v)
	}
	in := huh.NewInput().
		Title(fmt.Sprintf("%s › %s › %s", p.ServiceName, p.Type, p.Name)).
		Description(desc).
		Value(&plain)
	if p.Sensitive {
		in = in.EchoMode(huh.EchoModePassword)
	}
	ok, err := back(huh.NewForm(huh.NewGroup(in)))
	if err != nil || !ok {
		return "", false, err
	}
	nv := quoteValue(plain, quoted)
	return nv, nv != v, nil
}

// editText edits a value as multi-line text.
func editText(p client.AgentConfigProperty, v string) (string, bool, error) {
	text := v
	area := huh.NewText().
		Title(fmt.Sprintf("%s › %s › %s", p.ServiceName, p.Type, p.Name)).
		Description("Edit the value as YAML text. alt+enter or ctrl+j: new line · ctrl+e: open $EDITOR · enter: save · esc: back").
		Lines(min(max(strings.Count(v, "\n")+2, 6), 18)).
		EditorExtension("yaml").
		Value(&text)
	ok, err := back(huh.NewForm(huh.NewGroup(area)))
	if err != nil || !ok {
		return "", false, err
	}
	if strings.Contains(v, "\n") && text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	return text, text != v, nil
}

// review shows the pending changes. ok is true when the user chose to apply.
func (e *agentConfigEditor) review() (bool, error) {
	var b strings.Builder
	changes := e.changes()
	if len(changes) == 0 {
		b.WriteString("No changes.")
	}
	const maxLines = 24
	for n, ch := range changes {
		if n >= maxLines {
			fmt.Fprintf(&b, "... and %d more\n", len(changes)-n)
			break
		}
		var p client.AgentConfigProperty
		var i int
		for i = range e.props {
			if e.props[i].Key() == ch.Property {
				p = e.props[i]
				break
			}
		}
		nv, _ := e.value(i)
		suffix := ""
		if ch.Reset {
			suffix = " (reset to default)"
		}
		if lines, ok := entryDiff(p.Value, nv); ok && !p.Sensitive {
			fmt.Fprintf(&b, "%s › %s › %s%s\n", p.ServiceName, p.Type, p.Name, suffix)
			for _, l := range lines {
				fmt.Fprintf(&b, "    %s\n", truncate(l, 100))
			}
			continue
		}
		fmt.Fprintf(&b, "%s › %s › %s: %s → %s%s\n", p.ServiceName, p.Type, p.Name,
			previewValue(p.Value, p.Sensitive, 40), previewValue(nv, p.Sensitive, 40), suffix)
	}
	opts := []huh.Option[string]{huh.NewOption("Back to editing", choiceBack)}
	if len(changes) > 0 {
		opts = append([]huh.Option[string]{huh.NewOption(fmt.Sprintf("Apply %d change(s)", len(changes)), choiceApply)}, opts...)
	}
	opts = append(opts, huh.NewOption("Cancel", choiceCancel))
	choice := opts[0].Value
	sel := huh.NewSelect[string]().
		Title(e.title + " - review").
		Description(strings.TrimRight(b.String(), "\n")).
		Options(opts...).
		Value(&choice)
	if err := runForm(huh.NewForm(huh.NewGroup(sel))); err != nil {
		return false, err
	}
	switch choice {
	case choiceApply:
		return true, nil
	case choiceCancel:
		return false, ErrCanceled
	}
	return false, nil
}
