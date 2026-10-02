package cmd

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/config"
	"github.com/jbrazda/iics-cli/internal/output"
	"github.com/jbrazda/iics-cli/internal/tui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newUserCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "Manage users",
	}

	cmd.AddCommand(newUserListCmd())
	cmd.AddCommand(newUserGetCmd())
	cmd.AddCommand(newUserCreateCmd())
	cmd.AddCommand(newUserEditCmd(""))
	cmd.AddCommand(newUserEditCmd("update"))
	cmd.AddCommand(newUserMembershipCmd(userRoleMembership))
	cmd.AddCommand(newUserMembershipCmd(userGroupMembership))
	cmd.AddCommand(newUserDeleteCmd())
	cmd.AddCommand(newUserChangePasswordCmd())
	cmd.AddCommand(newUserResetPasswordCmd())
	return cmd
}

// ---------------------------------------------------------------------------
// user list
// ---------------------------------------------------------------------------

func newUserListCmd() *cobra.Command {
	var opts client.UserListOptions

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List users",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			users, err := c.ListUsers(context.Background(), opts)
			if err != nil {
				return err
			}

			f, err := getFormatter()
			if err != nil {
				return err
			}

			columns := []output.Column{
				{Header: "ID", Field: "id", Width: 24, Priority: 5, Shrink: output.ShrinkNever},
				{Header: "USERNAME", Field: "userName", Width: 30, Priority: 1},
				{Header: "EMAIL", Field: "email", Width: 35, Priority: 3, Shrink: output.ShrinkTruncate},
				{Header: "STATE", Field: "state", Width: 10, Priority: 2, Shrink: output.ShrinkNever},
				{Header: "AUTH", Field: "authentication", Width: 10, Priority: 2, Shrink: output.ShrinkNever},
				{Header: "UPDATED", Field: "updateTime", Width: 22, Priority: 2, Shrink: output.ShrinkNever},
			}

			return f.Format(users, columns)
		},
	}

	cmd.Flags().IntVar(&opts.Limit, "limit", 200, "max results")
	cmd.Flags().IntVar(&opts.Skip, "skip", 0, "number of results to skip")
	return cmd
}

// ---------------------------------------------------------------------------
// user get
// ---------------------------------------------------------------------------

// propRow is a two-column property/value row for the user details table.
type propRow struct {
	Property string `json:"property"`
	Value    string `json:"value"`
}

func userToPropRows(u *client.User) []propRow {
	boolStr := func(b bool) string {
		if b {
			return "true"
		}
		return "false"
	}
	return []propRow{
		{Property: "id", Value: u.ID},
		{Property: "orgId", Value: u.OrgID},
		{Property: "userName", Value: u.UserName},
		{Property: "firstName", Value: u.FirstName},
		{Property: "lastName", Value: u.LastName},
		{Property: "email", Value: u.Email},
		{Property: "phone", Value: u.Phone},
		{Property: "state", Value: u.State},
		{Property: "timeZoneId", Value: u.TimeZoneID},
		{Property: "title", Value: u.Title},
		{Property: "authentication", Value: u.Authentication},
		{Property: "lastLoginMode", Value: u.LastLoginMode},
		{Property: "lastLoginTime", Value: u.LastLoginTime},
		{Property: "maxLoginAttempts", Value: u.MaxLoginAttempts},
		{Property: "forcePasswordChange", Value: boolStr(u.ForcePasswordChange)},
		{Property: "description", Value: u.Description},
		{Property: "createTime", Value: u.CreateTime},
		{Property: "createdBy", Value: u.CreatedBy},
		{Property: "updateTime", Value: u.UpdateTime},
		{Property: "updatedBy", Value: u.UpdatedBy},
	}
}

// printUserSections renders the three-section table layout for a single user.
func printUserSections(w io.Writer, u *client.User, style output.TableStyle) error {
	tf := output.New(output.FormatTable, w, style)

	propCols := []output.Column{
		{Header: "PROPERTY", Field: "property", Width: 22},
		{Header: "VALUE", Field: "value"},
	}

	_, _ = fmt.Fprintln(w, "User Details:")
	_, _ = fmt.Fprintln(w)
	if err := tf.Format(userToPropRows(u), propCols); err != nil {
		return err
	}

	if len(u.Groups) > 0 {
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "Groups:")
		_, _ = fmt.Fprintln(w)
		groupCols := []output.Column{
			{Header: "ID", Field: "id", Width: 24},
			{Header: "NAME", Field: "userGroupName", Width: 25},
			{Header: "DESCRIPTION", Field: "description"},
		}
		if err := tf.Format(u.Groups, groupCols); err != nil {
			return err
		}
	}

	if len(u.Roles) > 0 {
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "Roles:")
		_, _ = fmt.Fprintln(w)
		roleCols := []output.Column{
			{Header: "ID", Field: "id", Width: 24},
			{Header: "NAME", Field: "displayName", Width: 30},
			{Header: "DESCRIPTION", Field: "displayDescription"},
		}
		if err := tf.Format(u.Roles, roleCols); err != nil {
			return err
		}
	}

	return nil
}

// buildUserCSVColumns constructs output columns for CSV export from a comma-separated
// list of field names. The special fields "groups" and "roles" are rendered as
// pipe-separated IDs.
func buildUserCSVColumns(fields string) []output.Column {
	knownCols := map[string]output.Column{
		"id":                  {Header: "id", Field: "id"},
		"orgId":               {Header: "orgId", Field: "orgId"},
		"userName":            {Header: "userName", Field: "userName"},
		"firstName":           {Header: "firstName", Field: "firstName"},
		"lastName":            {Header: "lastName", Field: "lastName"},
		"email":               {Header: "email", Field: "email"},
		"phone":               {Header: "phone", Field: "phone"},
		"title":               {Header: "title", Field: "title"},
		"description":         {Header: "description", Field: "description"},
		"state":               {Header: "state", Field: "state"},
		"authentication":      {Header: "authentication", Field: "authentication"},
		"timeZoneId":          {Header: "timeZoneId", Field: "timeZoneId"},
		"lastLoginTime":       {Header: "lastLoginTime", Field: "lastLoginTime"},
		"lastLoginMode":       {Header: "lastLoginMode", Field: "lastLoginMode"},
		"maxLoginAttempts":    {Header: "maxLoginAttempts", Field: "maxLoginAttempts"},
		"forcePasswordChange": {Header: "forcePasswordChange", Field: "forcePasswordChange"},
		"createTime":          {Header: "createTime", Field: "createTime"},
		"updateTime":          {Header: "updateTime", Field: "updateTime"},
		"createdBy":           {Header: "createdBy", Field: "createdBy"},
		"updatedBy":           {Header: "updatedBy", Field: "updatedBy"},
		"groups": {
			Header: "groups",
			Field:  "groups",
			Func: func(v interface{}) string {
				row, ok := v.(map[string]interface{})
				if !ok {
					return ""
				}
				return joinIDsFromSlice(row["groups"])
			},
		},
		"roles": {
			Header: "roles",
			Field:  "roles",
			Func: func(v interface{}) string {
				row, ok := v.(map[string]interface{})
				if !ok {
					return ""
				}
				return joinIDsFromSlice(row["roles"])
			},
		},
	}

	cols := make([]output.Column, 0)
	for _, f := range strings.Split(fields, ",") {
		f = strings.TrimSpace(f)
		if col, ok := knownCols[f]; ok {
			cols = append(cols, col)
		}
	}
	return cols
}

// joinIDsFromSlice renders a JSON array of objects as pipe-separated "id" values.
func joinIDsFromSlice(v interface{}) string {
	items, ok := v.([]interface{})
	if !ok {
		return ""
	}
	ids := make([]string, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if id, ok := m["id"].(string); ok && id != "" {
			ids = append(ids, id)
		}
	}
	return strings.Join(ids, "|")
}

const defaultCSVFields = "id,userName,firstName,lastName,email,state,authentication,lastLoginTime"

func newUserGetCmd() *cobra.Command {
	var (
		id        string
		userName  string
		csvFields string
	)

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get user details (searches interactively without --id/--username)",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := getClient(cmd)
			if err != nil {
				return err
			}

			var user *client.User
			switch {
			case id != "":
				user, err = c.GetUser(context.Background(), id)
			case userName != "":
				user, err = c.GetUserByName(context.Background(), userName)
			case isInteractiveTTY():
				user, err = promptUserSearch(context.Background(), c)
				if err == nil && user == nil {
					_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Canceled.")
					return nil
				}
			default:
				return fmt.Errorf("--id or --username is required")
			}
			if err != nil {
				return err
			}
			return printUser(user, csvFields)
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "user ID")
	cmd.Flags().StringVar(&userName, "username", "", "user name (exact match)")
	cmd.Flags().StringVar(&csvFields, "fields", defaultCSVFields, "comma-separated fields for CSV output")
	return cmd
}

// ---------------------------------------------------------------------------
// helpers shared by create / update
// ---------------------------------------------------------------------------

// buildGroupMap fetches all user groups (paginated) and returns a name->ID map.
func buildGroupMap(ctx context.Context, c *client.Client) (map[string]string, error) {
	m := make(map[string]string)
	opts := client.UserGroupListOptions{Limit: 200}
	for {
		batch, err := c.ListUserGroups(ctx, opts)
		if err != nil {
			return nil, fmt.Errorf("fetching groups: %w", err)
		}
		for _, g := range batch {
			m[g.UserGroupName] = g.ID
		}
		if len(batch) < opts.Limit {
			break
		}
		opts.Skip += opts.Limit
	}
	return m, nil
}

// buildRoleMap fetches all roles (paginated) and returns a name->ID map.
func buildRoleMap(ctx context.Context, c *client.Client) (map[string]string, error) {
	m := make(map[string]string)
	opts := client.RoleListOptions{Limit: 200}
	for {
		batch, err := c.ListRoles(ctx, opts)
		if err != nil {
			return nil, fmt.Errorf("fetching roles: %w", err)
		}
		for _, r := range batch {
			m[r.RoleName] = r.ID
		}
		if len(batch) < opts.Limit {
			break
		}
		opts.Skip += opts.Limit
	}
	return m, nil
}

// detectInputFormat infers the file format from extension or content.
func detectInputFormat(filename string, data []byte) string {
	if filename != "-" && filename != "" {
		switch strings.ToLower(filepath.Ext(filename)) {
		case ".json":
			return "json"
		case ".yaml", ".yml":
			return "yaml"
		case ".csv":
			return "csv"
		}
	}
	trimmed := strings.TrimSpace(string(data))
	if len(trimmed) > 0 {
		if trimmed[0] == '{' || trimmed[0] == '[' {
			return "json"
		}
		nl := strings.IndexByte(trimmed, '\n')
		firstLine := trimmed
		if nl > 0 {
			firstLine = trimmed[:nl]
		}
		if strings.Count(firstLine, ",") >= 2 {
			return "csv"
		}
	}
	return "yaml"
}

// parseUsersFromBytes decodes users from raw bytes. For CSV, group/role columns are
// expected to be pipe-separated names; they are stored in UserGroupRef.UserGroupName
// and UserRole.RoleName respectively (IDs must be resolved separately).
func parseUsersFromBytes(data []byte, format string) ([]client.User, error) {
	switch format {
	case "json":
		return parseUsersJSON(data)
	case "yaml":
		return parseUsersYAML(data)
	case "csv":
		return parseUsersCSV(data)
	default:
		return nil, fmt.Errorf("unsupported format %q", format)
	}
}

func parseUsersJSON(data []byte) ([]client.User, error) {
	trimmed := strings.TrimSpace(string(data))
	if strings.HasPrefix(trimmed, "[") {
		var users []client.User
		if err := json.Unmarshal(data, &users); err != nil {
			return nil, fmt.Errorf("parsing JSON array: %w", err)
		}
		return users, nil
	}
	var u client.User
	if err := json.Unmarshal(data, &u); err != nil {
		return nil, fmt.Errorf("parsing JSON object: %w", err)
	}
	return []client.User{u}, nil
}

func parseUsersYAML(data []byte) ([]client.User, error) {
	// Try array first
	var users []client.User
	if err := yaml.Unmarshal(data, &users); err == nil && len(users) > 0 {
		return users, nil
	}
	var u client.User
	if err := yaml.Unmarshal(data, &u); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}
	return []client.User{u}, nil
}

func parseUsersCSV(data []byte) ([]client.User, error) {
	r := csv.NewReader(strings.NewReader(string(data)))
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parsing CSV: %w", err)
	}
	if len(records) < 2 {
		return nil, fmt.Errorf("CSV must have a header row and at least one data row")
	}

	headers := records[0]
	colIdx := make(map[string]int, len(headers))
	for i, h := range headers {
		colIdx[strings.TrimSpace(h)] = i
	}

	get := func(row []string, col string) string {
		i, ok := colIdx[col]
		if !ok || i >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[i])
	}

	users := make([]client.User, 0, len(records)-1)
	for _, row := range records[1:] {
		u := client.User{
			UserName:       get(row, "userName"),
			FirstName:      get(row, "firstName"),
			LastName:       get(row, "lastName"),
			Email:          get(row, "email"),
			Phone:          get(row, "phone"),
			Title:          get(row, "title"),
			Description:    get(row, "description"),
			State:          get(row, "state"),
			TimeZoneID:     get(row, "timeZoneId"),
			Authentication: get(row, "authentication"),
		}
		if get(row, "forcePasswordChange") == "true" {
			u.ForcePasswordChange = true
		}
		// Groups: pipe-separated names stored in UserGroupName (no ID yet)
		if gs := get(row, "groups"); gs != "" {
			for _, name := range strings.Split(gs, "|") {
				name = strings.TrimSpace(name)
				if name != "" {
					u.Groups = append(u.Groups, client.UserGroupRef{UserGroupName: name})
				}
			}
		}
		// Roles: pipe-separated names stored in RoleName (no ID yet)
		if rs := get(row, "roles"); rs != "" {
			for _, name := range strings.Split(rs, "|") {
				name = strings.TrimSpace(name)
				if name != "" {
					u.Roles = append(u.Roles, client.UserRole{RoleName: name})
				}
			}
		}
		users = append(users, u)
	}
	return users, nil
}

// resolveUserGroupsAndRoles replaces name-only group/role references with real IDs.
// Fetches groups and roles from the API only when needed (lazy).
func resolveUserGroupsAndRoles(ctx context.Context, c *client.Client, users []client.User) error {
	needsGroups := false
	needsRoles := false
	for _, u := range users {
		for _, g := range u.Groups {
			if g.ID == "" {
				needsGroups = true
			}
		}
		for _, r := range u.Roles {
			if r.ID == "" {
				needsRoles = true
			}
		}
	}

	var groupMap, roleMap map[string]string
	if needsGroups {
		var err error
		groupMap, err = buildGroupMap(ctx, c)
		if err != nil {
			return err
		}
	}
	if needsRoles {
		var err error
		roleMap, err = buildRoleMap(ctx, c)
		if err != nil {
			return err
		}
	}

	for i := range users {
		for j := range users[i].Groups {
			g := &users[i].Groups[j]
			if g.ID == "" && g.UserGroupName != "" {
				id, ok := groupMap[g.UserGroupName]
				if !ok {
					return fmt.Errorf("group %q not found", g.UserGroupName)
				}
				g.ID = id
			}
		}
		for j := range users[i].Roles {
			r := &users[i].Roles[j]
			if r.ID == "" && r.RoleName != "" {
				id, ok := roleMap[r.RoleName]
				if !ok {
					return fmt.Errorf("role %q not found", r.RoleName)
				}
				r.ID = id
			}
		}
	}
	return nil
}

// runUserWizard interactively collects user fields in a paged form with a
// review step. When existing is non-nil, its values are pre-filled and the
// review shows the changes. Returns nil when the user cancels.
func runUserWizard(ctx context.Context, c *client.Client, existing *client.User) (*client.User, error) {
	if !config.IsTerminal() {
		return nil, fmt.Errorf("--interactive requires an interactive terminal")
	}

	u := &client.User{}
	if existing != nil {
		*u = *existing
	}

	groups, err := listAllUserGroups(ctx, c)
	if err != nil {
		return nil, fmt.Errorf("fetching groups: %w", err)
	}
	roles, err := c.ListRoles(ctx, client.RoleListOptions{})
	if err != nil {
		return nil, fmt.Errorf("fetching roles: %w", err)
	}

	cfg, _ := loadConfig()
	_, _, profileName, _ := resolveProfile()
	var prof *config.Profile
	if cfg != nil {
		prof = cfg.Profiles[profileName]
	}

	ok, err := tui.RunUserWizard(tui.UserWizardInput{
		User:        u,
		Update:      existing != nil,
		Groups:      groups,
		Roles:       roles,
		ProfileName: profileName,
		Patterns:    config.NewUserPatterns(cfg.GlobalNewUser(), prof),
		UserExists: func(name string) (string, error) {
			existing, lerr := c.FindUserByName(ctx, name)
			if lerr != nil || existing == nil {
				return "", lerr
			}
			return existing.ID, nil
		},
		Out:        os.Stderr,
		Accessible: prompter.Accessible,
	})
	if err != nil || !ok {
		return nil, err
	}
	return u, nil
}

// ---------------------------------------------------------------------------
// user create
// ---------------------------------------------------------------------------

type userCreateResult struct {
	ID        string `json:"id"`
	UserName  string `json:"userName"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
	State     string `json:"state"`
	Status    string `json:"status"`
	Elapsed   string `json:"elapsed,omitempty"`
}

func newUserCreateCmd() *cobra.Command {
	var (
		fromFile    string
		interactive bool
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create one or more users",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()

			// Interactive wizard: create a single user
			if interactive {
				u, wErr := runUserWizard(ctx, c, nil)
				if wErr != nil {
					return wErr
				}
				if u == nil {
					_, _ = fmt.Fprintln(os.Stderr, "Canceled.")
					return nil
				}
				created, cErr := c.CreateUser(ctx, u)
				if cErr != nil {
					return cErr
				}
				cfg, _ := loadConfig()
				return printUserSections(os.Stdout, created, resolveTableStyle(cfg))
			}

			if fromFile == "" {
				return fmt.Errorf("--from-file or --interactive is required")
			}

			var data []byte
			if fromFile == "-" {
				data, err = io.ReadAll(os.Stdin)
			} else {
				data, err = os.ReadFile(fromFile)
			}
			if err != nil {
				return fmt.Errorf("reading input: %w", err)
			}

			format := detectInputFormat(fromFile, data)
			users, err := parseUsersFromBytes(data, format)
			if err != nil {
				return err
			}
			if resolveErr := resolveUserGroupsAndRoles(ctx, c, users); resolveErr != nil {
				return resolveErr
			}

			// Single user: display like user get
			if len(users) == 1 {
				created, cErr := c.CreateUser(ctx, &users[0])
				if cErr != nil {
					return cErr
				}
				cfg, _ := loadConfig()
				style := resolveTableStyle(cfg)
				f, fErr := output.ParseFormat(outputFmt)
				if fErr != nil {
					return fErr
				}
				if f == output.FormatTable {
					return printUserSections(os.Stdout, created, style)
				}
				fmtr := output.New(f, os.Stdout, style)
				return fmtr.Format(created, nil)
			}

			// Bulk creation
			results := make([]userCreateResult, 0, len(users))
			for i := range users {
				start := time.Now()
				created, cErr := c.CreateUser(ctx, &users[i])
				elapsed := time.Since(start).Round(time.Millisecond).String()

				res := userCreateResult{
					UserName:  users[i].UserName,
					FirstName: users[i].FirstName,
					LastName:  users[i].LastName,
					Email:     users[i].Email,
					Elapsed:   elapsed,
				}
				if cErr != nil {
					res.Status = "Error: " + cErr.Error()
					_, _ = fmt.Fprintf(os.Stderr, "Error creating %s: %v\n", users[i].UserName, cErr)
				} else {
					res.ID = created.ID
					res.State = created.State
					res.Status = "Success"
					if verbose {
						slog.Info("user created",
							"userName", created.UserName,
							"id", created.ID,
							"status", "Success",
							"elapsed", elapsed)
					}
				}
				results = append(results, res)
			}

			f, err := getFormatter()
			if err != nil {
				return err
			}
			cols := []output.Column{
				{Header: "ID", Field: "id", Width: 24},
				{Header: "USERNAME", Field: "userName", Width: 30},
				{Header: "FIRST", Field: "firstName", Width: 15},
				{Header: "LAST", Field: "lastName", Width: 15},
				{Header: "EMAIL", Field: "email", Width: 30},
				{Header: "STATE", Field: "state", Width: 10},
				{Header: "STATUS", Field: "status", Width: 20},
			}
			return f.Format(results, cols)
		},
	}

	cmd.Flags().StringVar(&fromFile, "from-file", "", "JSON, YAML or CSV file with user(s); use - for stdin")
	cmd.Flags().BoolVar(&interactive, "interactive", false, "launch interactive creation wizard")
	return cmd
}

// ---------------------------------------------------------------------------
// user edit
// ---------------------------------------------------------------------------

// newUserEditCmd builds "user edit". With deprecatedUse set it builds the
// hidden, deprecated alias of that name (formerly "user update").
func newUserEditCmd(deprecatedUse string) *cobra.Command {
	var (
		id          string
		userName    string
		interactive bool
	)

	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Edit a user's group and role assignments (interactive)",
		Long: `Edit a user's user group and role assignments in an interactive form.

The IICS API cannot update user properties (name, email, title, time zone),
so only group and role assignments can be changed. The form shows the user's
current assignments pre-checked and ends with a review before any change is
applied. Requires a terminal; for scripts use "user update-roles" and
"user update-groups".`,
		Example: `  iics user edit --username jdoe@example.com
  iics user edit            # search for the user first`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !isInteractiveTTY() {
				return fmt.Errorf("user edit requires a terminal; use 'iics user update-roles' or 'iics user update-groups' in scripts")
			}
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()

			target, err := resolveUser(ctx, c, id, userName)
			if err != nil {
				return err
			}
			if target == nil {
				return nil // canceled
			}
			updated, err := runUserWizard(ctx, c, target)
			if err != nil {
				return err
			}
			if updated == nil {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Canceled.")
				return nil
			}
			if err = applyUserMembership(ctx, c, cmd.ErrOrStderr(), target, updated); err != nil {
				return err
			}
			result, err := c.GetUser(ctx, target.ID)
			if err != nil {
				return err
			}
			return printUser(result, defaultCSVFields)
		},
	}
	if deprecatedUse != "" {
		cmd.Use = deprecatedUse
		cmd.Deprecated = "use 'iics user edit' (interactive) or 'iics user update-roles' / 'update-groups' (scripts)"
		cmd.Hidden = true
	}

	cmd.Flags().StringVar(&id, "id", "", "user ID")
	cmd.Flags().StringVar(&userName, "username", "", "user name (exact match)")
	cmd.MarkFlagsMutuallyExclusive("id", "username")
	// Kept so older invocations ("--interactive") still parse; edit is always
	// interactive.
	cmd.Flags().BoolVar(&interactive, "interactive", false, "no longer needed; always interactive")
	_ = cmd.Flags().MarkHidden("interactive")
	return cmd
}

// ---------------------------------------------------------------------------
// user delete
// ---------------------------------------------------------------------------

func newUserDeleteCmd() *cobra.Command {
	var (
		id       string
		userName string
		yes      bool
	)

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a user",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()

			target, err := resolveUser(ctx, c, id, userName)
			if err != nil {
				return err
			}
			if target == nil {
				return nil // canceled
			}

			if !yes {
				_, _ = fmt.Fprintf(os.Stderr,
					"Are you sure you want to delete user %s (%s)? [y/N]: ",
					target.UserName, target.ID)
				var confirm string
				_, _ = fmt.Scanln(&confirm)
				if confirm != "y" && confirm != "Y" {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Canceled.")
					return nil
				}
			}

			if verbose {
				slog.Info("deleting user", "id", target.ID, "userName", target.UserName)
			}
			if err := c.DeleteUser(ctx, target.ID); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "User deleted: %s (%s)\n", target.UserName, target.ID)
			return nil
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "user ID")
	cmd.Flags().StringVar(&userName, "username", "", "user name (exact match)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation prompt")
	return cmd
}

// ---------------------------------------------------------------------------
// user change-password
// ---------------------------------------------------------------------------

func newUserChangePasswordCmd() *cobra.Command {
	var (
		newPassword string
		oldPassword string
		id          string
		userName    string
	)

	cmd := &cobra.Command{
		Use:   "change-password",
		Short: "Change a user password",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()

			req := &client.ChangePasswordRequest{}

			// Determine whether this is an admin (--id/--username) or own-password change.
			isAdmin := id != "" || userName != ""

			if isAdmin {
				target, rErr := resolveUser(ctx, c, id, userName)
				if rErr != nil {
					return rErr
				}
				if target == nil {
					return nil
				}
				req.UserID = target.ID
			}

			// Non-TTY: all required flags must be present.
			if !config.IsTerminal() {
				if !isAdmin && oldPassword == "" {
					return fmt.Errorf("--old-password is required when not using --id or --username")
				}
				if newPassword == "" {
					return fmt.Errorf("--new-password is required")
				}
				req.OldPassword = oldPassword
				req.NewPassword = newPassword
				if cpErr := c.ChangePassword(ctx, req); cpErr != nil {
					return cpErr
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Password changed successfully.")
				return nil
			}

			// TTY: prompt for any missing inputs.
			if !isAdmin && oldPassword == "" {
				oldPassword, err = promptPassword("Current password: ")
				if err != nil {
					return err
				}
			}
			req.OldPassword = oldPassword

			if newPassword == "" {
				newPassword, err = promptPasswordConfirm("New password")
				if err != nil {
					return err
				}
				if newPassword == "" {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Canceled.")
					return nil
				}
			}
			req.NewPassword = newPassword

			if err := c.ChangePassword(ctx, req); err != nil {
				return err
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Password changed successfully.")
			return nil
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "user ID (admin: change another user's password)")
	cmd.Flags().StringVar(&userName, "username", "", "user name (admin: change another user's password)")
	cmd.Flags().StringVar(&oldPassword, "old-password", "", "current password (required when changing own password without interactive prompt)")
	cmd.Flags().StringVar(&newPassword, "new-password", "", "new password (skips interactive prompt)")
	return cmd
}

// ---------------------------------------------------------------------------
// user reset-password
// ---------------------------------------------------------------------------

func newUserResetPasswordCmd() *cobra.Command {
	var (
		id             string
		userName       string
		securityAnswer string
		newPassword    string
	)

	cmd := &cobra.Command{
		Use:   "reset-password",
		Short: "Reset a user password using the security answer",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()

			// Fully inlined (all flags provided): call API immediately.
			if (id != "" || userName != "") && securityAnswer != "" && newPassword != "" {
				target, rErr := resolveUser(ctx, c, id, userName)
				if rErr != nil {
					return rErr
				}
				if target == nil {
					return nil
				}
				if target.Authentication != "Native" {
					return fmt.Errorf("reset-password is only supported for Native authentication users (user %s uses %s)",
						target.UserName, target.Authentication)
				}
				req := &client.ResetPasswordRequest{
					UserID:         target.ID,
					SecurityAnswer: securityAnswer,
					NewPassword:    newPassword,
				}
				if rpErr := c.ResetPassword(ctx, req); rpErr != nil {
					return rpErr
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Password reset successfully.")
				return nil
			}

			// Non-TTY with partial flags: return usage error.
			if !config.IsTerminal() {
				missing := []string{}
				if id == "" && userName == "" {
					missing = append(missing, "--id or --username")
				}
				if securityAnswer == "" {
					missing = append(missing, "--security-answer")
				}
				if newPassword == "" {
					missing = append(missing, "--new-password")
				}
				return fmt.Errorf("non-interactive mode requires: %s", strings.Join(missing, ", "))
			}

			// TTY: interactive loop.
			for {
				// Resolve target user if not already identified.
				var target *client.User
				if id != "" || userName != "" {
					target, err = resolveUser(ctx, c, id, userName)
					if err != nil {
						return err
					}
				} else {
					target, err = promptUserSearch(ctx, c)
					if err != nil {
						return err
					}
				}
				if target == nil {
					return nil // canceled
				}

				if target.Authentication != "Native" {
					_, _ = fmt.Fprintf(os.Stderr,
						"Error: reset-password is only supported for Native authentication users.\n"+
							"User %s uses %s authentication.\n",
						target.UserName, target.Authentication)
					if id != "" || userName != "" {
						return fmt.Errorf("user %s does not use Native authentication", target.UserName)
					}
					// Allow re-search
					id = ""
					userName = ""
					continue
				}

				// Security answer
				if securityAnswer == "" {
					securityAnswer, err = promptPassword("Security Answer: ")
					if err != nil {
						return err
					}
					if securityAnswer == "" {
						return nil // canceled
					}
				}

				// New password + confirmation
				if newPassword == "" {
					newPassword, err = promptPasswordConfirm("New password")
					if err != nil {
						return err
					}
					if newPassword == "" {
						return nil // canceled
					}
				}

				req := &client.ResetPasswordRequest{
					UserID:         target.ID,
					SecurityAnswer: securityAnswer,
					NewPassword:    newPassword,
				}
				if err := c.ResetPassword(ctx, req); err != nil {
					return err
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Password reset successfully.")
				return nil
			}
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "user ID")
	cmd.Flags().StringVar(&userName, "username", "", "user name (exact match)")
	cmd.Flags().StringVar(&securityAnswer, "security-answer", "", "answer to the security question")
	cmd.Flags().StringVar(&newPassword, "new-password", "", "new password (skips interactive prompt)")
	return cmd
}
