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
	cmd.AddCommand(newUserUpdateCmd())
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
				{Header: "ID", Field: "id", Width: 24},
				{Header: "USERNAME", Field: "userName", Width: 30},
				{Header: "EMAIL", Field: "email", Width: 35},
				{Header: "STATE", Field: "state", Width: 10},
				{Header: "AUTH", Field: "authentication", Width: 10},
				{Header: "UPDATED", Field: "updateTime", Width: 22},
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
		Short: "Get user details",
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
			default:
				return fmt.Errorf("--id or --username is required")
			}
			if err != nil {
				return err
			}

			f, err := output.ParseFormat(outputFmt)
			if err != nil {
				return err
			}

			cfg, _ := loadConfig()
			style := resolveTableStyle(cfg)

			switch f {
			case output.FormatCSV:
				cols := buildUserCSVColumns(csvFields)
				csvFmt := output.New(output.FormatCSV, os.Stdout, style)
				return csvFmt.Format([]*client.User{user}, cols)
			case output.FormatTable:
				return printUserSections(os.Stdout, user, style)
			default:
				// JSON / YAML: output the full struct
				fmtr := output.New(f, os.Stdout, style)
				return fmtr.Format(user, nil)
			}
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

// buildGroupMap fetches all user groups and returns a name->ID map.
func buildGroupMap(ctx context.Context, c *client.Client) (map[string]string, error) {
	groups, err := c.ListUserGroups(ctx, client.UserGroupListOptions{Limit: 1000})
	if err != nil {
		return nil, fmt.Errorf("fetching groups: %w", err)
	}
	m := make(map[string]string, len(groups))
	for _, g := range groups {
		m[g.UserGroupName] = g.ID
	}
	return m, nil
}

// buildRoleMap fetches all roles and returns a name->ID map.
func buildRoleMap(ctx context.Context, c *client.Client) (map[string]string, error) {
	roles, err := c.ListRoles(ctx, client.RoleListOptions{Limit: 1000})
	if err != nil {
		return nil, fmt.Errorf("fetching roles: %w", err)
	}
	m := make(map[string]string, len(roles))
	for _, r := range roles {
		m[r.Name] = r.ID
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

// runUserWizard interactively collects user fields. When existing is non-nil, current
// values are shown as defaults. Returns the completed User.
func runUserWizard(ctx context.Context, c *client.Client, existing *client.User) (*client.User, error) {
	if !config.IsTerminal() {
		return nil, fmt.Errorf("--interactive requires an interactive terminal")
	}

	u := &client.User{}
	if existing != nil {
		*u = *existing
	}

	_, _ = fmt.Fprintln(os.Stderr)

	// Authentication type
	authIdx, err := promptSelect("Authentication type", []string{"Native", "SSO"})
	if err != nil {
		return nil, err
	}
	if authIdx < 0 {
		return nil, fmt.Errorf("canceled")
	}
	authTypes := []string{"Native", "SSO"}
	u.Authentication = authTypes[authIdx]

	// Scalar fields
	if u.FirstName, err = promptText("First Name", u.FirstName); err != nil {
		return nil, err
	}
	if u.LastName, err = promptText("Last Name", u.LastName); err != nil {
		return nil, err
	}

	// Username default: firstname.lastname@domain
	if u.UserName == "" && u.FirstName != "" && u.LastName != "" {
		_, p, _, _ := resolveProfile()
		domain := ""
		if p != nil && strings.Contains(p.Username, "@") {
			domain = p.Username[strings.Index(p.Username, "@")+1:]
		}
		if domain != "" {
			u.UserName = fmt.Sprintf("%s.%s@%s",
				strings.ToLower(u.FirstName),
				strings.ToLower(u.LastName),
				domain)
		}
	}
	if u.UserName, err = promptText("User Name", u.UserName); err != nil {
		return nil, err
	}
	if u.UserName == "" {
		return nil, fmt.Errorf("userName is required")
	}

	if u.Email, err = promptText("Email", u.Email); err != nil {
		return nil, err
	}
	if u.Phone, err = promptText("Phone (optional)", u.Phone); err != nil {
		return nil, err
	}
	if u.Title, err = promptText("Title (optional)", u.Title); err != nil {
		return nil, err
	}
	if u.Description, err = promptText("Description (optional)", u.Description); err != nil {
		return nil, err
	}
	if u.TimeZoneID, err = promptText("Time Zone ID (optional, e.g. America/New_York)", u.TimeZoneID); err != nil {
		return nil, err
	}

	// State
	stateDefault := 0
	if u.State == "Disabled" {
		stateDefault = 1
	}
	stateOptions := []string{"Active", "Disabled"}
	sIdx, err := promptSelect("State", stateOptions)
	if err != nil {
		return nil, err
	}
	if sIdx >= 0 {
		u.State = stateOptions[sIdx]
	} else if u.State == "" {
		u.State = stateOptions[stateDefault]
	}

	// Force password change
	forceChange, err := promptYesNo("Force password change on next login?", u.ForcePasswordChange)
	if err != nil {
		return nil, err
	}
	u.ForcePasswordChange = forceChange

	// Groups
	allGroups, err := c.ListUserGroups(ctx, client.UserGroupListOptions{Limit: 1000})
	if err != nil {
		return nil, fmt.Errorf("fetching groups: %w", err)
	}
	if len(allGroups) > 0 {
		groupOpts := make([]string, len(allGroups))
		groupDefaults := make([]int, 0)
		currentGroupIDs := make(map[string]bool)
		for _, g := range u.Groups {
			currentGroupIDs[g.ID] = true
		}
		for i, g := range allGroups {
			groupOpts[i] = fmt.Sprintf("%s - %s", g.UserGroupName, g.Description)
			if currentGroupIDs[g.ID] {
				groupDefaults = append(groupDefaults, i)
			}
		}
		selected, sErr := promptMultiSelect("Groups", groupOpts, groupDefaults)
		if sErr != nil {
			return nil, sErr
		}
		u.Groups = nil
		for _, idx := range selected {
			g := allGroups[idx]
			u.Groups = append(u.Groups, client.UserGroupRef{
				ID:            g.ID,
				UserGroupName: g.UserGroupName,
			})
		}
	}

	// Roles
	allRoles, err := c.ListRoles(ctx, client.RoleListOptions{Limit: 1000})
	if err != nil {
		return nil, fmt.Errorf("fetching roles: %w", err)
	}
	if len(allRoles) > 0 {
		roleOpts := make([]string, len(allRoles))
		roleDefaults := make([]int, 0)
		currentRoleIDs := make(map[string]bool)
		for _, r := range u.Roles {
			currentRoleIDs[r.ID] = true
		}
		for i, r := range allRoles {
			roleOpts[i] = fmt.Sprintf("%s - %s", r.Name, r.Description)
			if currentRoleIDs[r.ID] {
				roleDefaults = append(roleDefaults, i)
			}
		}
		selected, sErr := promptMultiSelect("Roles", roleOpts, roleDefaults)
		if sErr != nil {
			return nil, sErr
		}
		u.Roles = nil
		for _, idx := range selected {
			r := allRoles[idx]
			u.Roles = append(u.Roles, client.UserRole{
				ID:       r.ID,
				RoleName: r.Name,
			})
		}
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
// user update
// ---------------------------------------------------------------------------

func newUserUpdateCmd() *cobra.Command {
	var (
		id          string
		userName    string
		fromFile    string
		interactive bool
	)

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update one or more users",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()

			// Interactive update
			if interactive {
				target, rErr := resolveUser(ctx, c, id, userName)
				if rErr != nil {
					return rErr
				}
				if target == nil {
					return nil // user canceled
				}
				updated, wErr := runUserWizard(ctx, c, target)
				if wErr != nil {
					return wErr
				}
				result, uErr := c.UpdateUser(ctx, target.ID, updated)
				if uErr != nil {
					return uErr
				}
				cfg, _ := loadConfig()
				return printUserSections(os.Stdout, result, resolveTableStyle(cfg))
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
			if err := resolveUserGroupsAndRoles(ctx, c, users); err != nil {
				return err
			}

			// Single user: resolve target by --id/--username or by userName in file
			if len(users) == 1 {
				targetID := id
				if targetID == "" && users[0].ID != "" {
					targetID = users[0].ID
				}
				if targetID == "" && users[0].UserName != "" {
					target, lErr := c.GetUserByName(ctx, users[0].UserName)
					if lErr != nil {
						return lErr
					}
					targetID = target.ID
				}
				if targetID == "" {
					return fmt.Errorf("cannot determine user ID; provide --id or include id/userName in the file")
				}
				result, uErr := c.UpdateUser(ctx, targetID, &users[0])
				if uErr != nil {
					return uErr
				}
				cfg, _ := loadConfig()
				style := resolveTableStyle(cfg)
				f, fErr := output.ParseFormat(outputFmt)
				if fErr != nil {
					return fErr
				}
				if f == output.FormatTable {
					return printUserSections(os.Stdout, result, style)
				}
				fmtr := output.New(f, os.Stdout, style)
				return fmtr.Format(result, nil)
			}

			// Bulk update
			results := make([]userCreateResult, 0, len(users))
			for i := range users {
				start := time.Now()
				targetID := users[i].ID
				if targetID == "" && users[i].UserName != "" {
					target, lErr := c.GetUserByName(ctx, users[i].UserName)
					if lErr != nil {
						elapsed := time.Since(start).Round(time.Millisecond).String()
						results = append(results, userCreateResult{
							UserName: users[i].UserName,
							Status:   "Error: " + lErr.Error(),
							Elapsed:  elapsed,
						})
						_, _ = fmt.Fprintf(os.Stderr, "Error looking up %s: %v\n", users[i].UserName, lErr)
						continue
					}
					targetID = target.ID
				}
				if targetID == "" {
					results = append(results, userCreateResult{
						UserName: users[i].UserName,
						Status:   "Error: no id or userName to identify user",
					})
					continue
				}

				result, uErr := c.UpdateUser(ctx, targetID, &users[i])
				elapsed := time.Since(start).Round(time.Millisecond).String()

				res := userCreateResult{
					UserName:  users[i].UserName,
					FirstName: users[i].FirstName,
					LastName:  users[i].LastName,
					Email:     users[i].Email,
					Elapsed:   elapsed,
				}
				if uErr != nil {
					res.Status = "Error: " + uErr.Error()
					_, _ = fmt.Fprintf(os.Stderr, "Error updating %s: %v\n", users[i].UserName, uErr)
				} else {
					res.ID = result.ID
					res.State = result.State
					res.Status = "Success"
					if verbose {
						slog.Info("user updated",
							"userName", result.UserName,
							"id", result.ID,
							"elapsed", elapsed)
					}
				}
				results = append(results, res)
			}

			f, fErr := getFormatter()
			if fErr != nil {
				return fErr
			}
			cols := []output.Column{
				{Header: "ID", Field: "id", Width: 24},
				{Header: "USERNAME", Field: "userName", Width: 30},
				{Header: "STATE", Field: "state", Width: 10},
				{Header: "STATUS", Field: "status", Width: 20},
			}
			return f.Format(results, cols)
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "user ID of the user to update")
	cmd.Flags().StringVar(&userName, "username", "", "user name of the user to update")
	cmd.Flags().StringVar(&fromFile, "from-file", "", "JSON, YAML or CSV file with updated user(s); use - for stdin")
	cmd.Flags().BoolVar(&interactive, "interactive", false, "interactively edit user fields")
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
