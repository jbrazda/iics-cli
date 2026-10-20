package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/config"
	"github.com/jbrazda/iics-cli/internal/tui"
)

// pickFilterThreshold is the list size above which the pickers first ask for
// a name filter.
const pickFilterThreshold = 10

// listProjects returns all projects from the objects API.
func listProjects(ctx context.Context, c *client.Client) ([]client.Object, error) {
	res, err := c.ListAllObjects(ctx, client.ObjectsListOptions{Type: "Project"}, nil)
	if err != nil {
		return nil, err
	}
	return sortedObjects(res.Objects), nil
}

// listFolders returns the folders of a project from the objects API.
func listFolders(ctx context.Context, c *client.Client, projectName string) ([]client.Object, error) {
	res, err := c.ListAllObjects(ctx, client.ObjectsListOptions{Query: fmt.Sprintf("location=='%s'", projectName)}, nil)
	if err != nil {
		return nil, err
	}
	var folders []client.Object
	for _, o := range sortedObjects(res.Objects) {
		if o.Type == "Folder" {
			folders = append(folders, o)
		}
	}
	return folders, nil
}

// objectName returns the last path segment of an object path.
func objectName(o client.Object) string {
	return o.Path[strings.LastIndex(o.Path, "/")+1:]
}

// pickByName lets the operator choose one object. Long lists first ask for a
// case-insensitive name filter (empty lists everything). Returns nil when
// canceled.
func pickByName(kind string, objs []client.Object) (*client.Object, error) {
	if len(objs) == 0 {
		return nil, fmt.Errorf("no %ss found", kind)
	}
	for {
		matches := objs
		filter := ""
		if len(objs) > pickFilterThreshold {
			text, err := promptText(fmt.Sprintf("Filter %ss by name (empty to list all %d)", kind, len(objs)), "")
			if err != nil {
				return nil, cancelAsNil(err)
			}
			filter = strings.TrimSpace(text)
			if filter != "" {
				matches = nil
				needle := strings.ToLower(filter)
				for _, o := range objs {
					if strings.Contains(strings.ToLower(objectName(o)), needle) {
						matches = append(matches, o)
					}
				}
			}
		}
		switch len(matches) {
		case 0:
			_, _ = fmt.Fprintf(os.Stderr, "No %ss match %q.\n", kind, filter)
			continue
		case 1:
			if filter != "" {
				return &matches[0], nil
			}
		}
		idx, err := tui.PickOne(prompter, fmt.Sprintf("Select a %s", kind), matches, objectName)
		if err != nil {
			return nil, cancelAsNil(err)
		}
		if idx < 0 {
			return nil, nil
		}
		return &matches[idx], nil
	}
}

// cancelAsNil turns a prompt cancellation into a nil error.
func cancelAsNil(err error) error {
	if errors.Is(err, tui.ErrCanceled) {
		return nil
	}
	return err
}

// pickProject lists the projects and lets the operator choose one.
// Returns nil when canceled.
func pickProject(ctx context.Context, c *client.Client) (*client.Object, error) {
	projects, err := listProjects(ctx, c)
	if err != nil {
		return nil, err
	}
	return pickByName("project", projects)
}

// pickFolder lets the operator choose a project, then one of its folders.
// Returns nil objects when canceled.
func pickFolder(ctx context.Context, c *client.Client) (project, folder *client.Object, err error) {
	project, err = pickProject(ctx, c)
	if err != nil || project == nil {
		return nil, nil, err
	}
	folders, err := listFolders(ctx, c, project.Path)
	if err != nil {
		return nil, nil, err
	}
	if len(folders) == 0 {
		return nil, nil, fmt.Errorf("project %q has no folders", project.Path)
	}
	folder, err = pickByName("folder", folders)
	if err != nil || folder == nil {
		return nil, nil, err
	}
	return project, folder, nil
}

// findObject returns the object in objs whose ID is id, or whose name is
// name; nil when not found.
func findObject(objs []client.Object, id, name string) *client.Object {
	for i, o := range objs {
		if (id != "" && o.ID == id) || (name != "" && strings.EqualFold(objectName(o), name)) {
			return &objs[i]
		}
	}
	return nil
}

// projectFolderEdit holds the values entered in the interactive editor.
// Only fields that differ from the current values are set.
type projectFolderEdit struct {
	Name        string
	Description string
}

// promptProjectFolderEdit asks for a new name and description, pre-filled
// with the current values (flag values win when given). ok is false when
// nothing changed or the operator declined.
func promptProjectFolderEdit(kind, target, curName, curDesc, flagName, flagDesc string) (edit projectFolderEdit, ok bool, err error) {
	if !config.IsTerminal() {
		return edit, false, fmt.Errorf("--interactive requires a terminal")
	}
	defName, defDesc := curName, curDesc
	if flagName != "" {
		defName = flagName
	}
	if flagDesc != "" {
		defDesc = flagDesc
	}
	newName, err := promptText(fmt.Sprintf("New %s name", kind), defName)
	if err != nil {
		return edit, false, cancelAsNil(err)
	}
	newDesc, err := promptText("Description (cannot be cleared)", defDesc)
	if err != nil {
		return edit, false, cancelAsNil(err)
	}
	newName, newDesc = strings.TrimSpace(newName), strings.TrimSpace(newDesc)
	if newName != "" && newName != curName {
		edit.Name = newName
	}
	if newDesc != "" && newDesc != curDesc {
		edit.Description = newDesc
	}
	if edit.Name == "" && edit.Description == "" {
		_, _ = fmt.Fprintln(os.Stderr, "Nothing to update.")
		return edit, false, nil
	}

	var changes []string
	if edit.Name != "" {
		changes = append(changes, fmt.Sprintf("name %q -> %q", curName, edit.Name))
	}
	if edit.Description != "" {
		changes = append(changes, fmt.Sprintf("description -> %q", edit.Description))
	}
	yes, err := promptYesNo(fmt.Sprintf("Update %s %s: %s?", kind, target, strings.Join(changes, ", ")), true)
	if err != nil {
		return edit, false, cancelAsNil(err)
	}
	if !yes {
		_, _ = fmt.Fprintln(os.Stderr, "Canceled.")
		return edit, false, nil
	}
	return edit, true, nil
}
