package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// frsAPIPath is the path of the internal FRS repository API used by the IICS
// web UI. It is undocumented; callers must fall back to the objects API.
const frsAPIPath = "frs/api/v1"

// frsPageSize is the $top page size for FRS collection requests.
const frsPageSize = 1000

// ContainerInfo describes a project or folder. CreatedTime and CreatedBy are
// only available from the FRS API; they are empty when the objects API was
// used.
type ContainerInfo struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Name        string `json:"name"`
	Path        string `json:"path"`
	Description string `json:"description"`
	CreatedTime string `json:"createTime,omitempty"`
	CreatedBy   string `json:"createdBy,omitempty"`
	UpdateTime  string `json:"updateTime,omitempty"`
	UpdatedBy   string `json:"updatedBy,omitempty"`
}

// ContainerList is the result of a project or folder listing. FRSError is set
// when the FRS API failed and the objects API was used instead.
type ContainerList struct {
	Items    []ContainerInfo
	FRSError error
}

// frsContainer is a project or folder as returned by the FRS API.
type frsContainer struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	CreatedBy       string `json:"createdBy"`
	LastUpdatedBy   string `json:"lastUpdatedBy"`
	CreatedTime     string `json:"createdTime"`
	LastUpdatedTime string `json:"lastUpdatedTime"`
	DocumentType    string `json:"documentType"`
	ParentInfo      []struct {
		ParentID   string `json:"parentId"`
		ParentName string `json:"parentName"`
		ParentType string `json:"parentType"`
	} `json:"parentInfo"`
}

type frsCollection struct {
	Value []frsContainer `json:"value"`
}

// ListProjectInfo lists all projects with created and updated details.
func (c *Client) ListProjectInfo(ctx context.Context) (*ContainerList, error) {
	items, frsErr := c.listFRSContainers(ctx, "Projects")
	if frsErr == nil {
		return &ContainerList{Items: items}, nil
	}
	res, err := c.ListAllObjects(ctx, ObjectsListOptions{Type: "Project"}, nil)
	if err != nil {
		return nil, err
	}
	return &ContainerList{Items: containersFromObjects(res.Objects), FRSError: frsErr}, nil
}

// ListFolderInfo lists folders with created and updated details: all folders
// when projectID and projectName are empty, otherwise the folders of that
// project (projectID wins).
func (c *Client) ListFolderInfo(ctx context.Context, projectID, projectName string) (*ContainerList, error) {
	collection := "Folders"
	if projectID != "" {
		collection = fmt.Sprintf("Projects('%s')/Folders", projectID)
	}
	items, frsErr := c.listFRSContainers(ctx, collection)
	if frsErr == nil {
		if projectID == "" && projectName != "" {
			items = filterByProject(items, projectName)
		}
		return &ContainerList{Items: items}, nil
	}

	opts := ObjectsListOptions{Type: "Folder"}
	if projectID != "" && projectName == "" {
		projects, err := c.ListAllObjects(ctx, ObjectsListOptions{Type: "Project"}, nil)
		if err != nil {
			return nil, err
		}
		for _, p := range projects.Objects {
			if p.ID == projectID {
				projectName = p.Path
			}
		}
		if projectName == "" {
			return nil, fmt.Errorf("project %s not found", projectID)
		}
	}
	if projectName != "" {
		opts = ObjectsListOptions{Query: fmt.Sprintf("location=='%s'", projectName)}
	}
	res, err := c.ListAllObjects(ctx, opts, nil)
	if err != nil {
		return nil, err
	}
	var folders []Object
	for _, o := range res.Objects {
		if o.Type == "Folder" {
			folders = append(folders, o)
		}
	}
	return &ContainerList{Items: containersFromObjects(folders), FRSError: frsErr}, nil
}

// FilterContainersByPath keeps the items whose path matches the regular
// expression pattern (case-insensitive). An empty pattern keeps all items.
func FilterContainersByPath(items []ContainerInfo, pattern string) ([]ContainerInfo, error) {
	if pattern == "" {
		return items, nil
	}
	re, err := regexp.Compile("(?i)" + pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid name pattern: %w", err)
	}
	var out []ContainerInfo
	for _, it := range items {
		if re.MatchString(it.Path) {
			out = append(out, it)
		}
	}
	return out, nil
}

func filterByProject(items []ContainerInfo, projectName string) []ContainerInfo {
	prefix := strings.ToLower(projectName) + "/"
	var out []ContainerInfo
	for _, it := range items {
		if strings.HasPrefix(strings.ToLower(it.Path), prefix) {
			out = append(out, it)
		}
	}
	return out
}

func containersFromObjects(objs []Object) []ContainerInfo {
	out := make([]ContainerInfo, 0, len(objs))
	for _, o := range objs {
		out = append(out, ContainerInfo{
			ID:          o.ID,
			Type:        o.Type,
			Name:        o.Path[strings.LastIndex(o.Path, "/")+1:],
			Path:        o.Path,
			Description: o.Description,
			UpdateTime:  o.UpdateTime,
			UpdatedBy:   o.UpdatedBy,
		})
	}
	return out
}

// listFRSContainers reads an FRS collection page by page and resolves the
// created/updated user IDs to user names.
func (c *Client) listFRSContainers(ctx context.Context, collection string) ([]ContainerInfo, error) {
	var all []frsContainer
	for skip := 0; ; skip += frsPageSize {
		var page frsCollection
		query := map[string]string{"$top": strconv.Itoa(frsPageSize)}
		if skip > 0 {
			query["$skip"] = strconv.Itoa(skip)
		}
		if err := c.doFRSJSON(ctx, collection, query, &page); err != nil {
			return nil, fmt.Errorf("FRS %s: %w", collection, err)
		}
		all = append(all, page.Value...)
		if len(page.Value) < frsPageSize {
			break
		}
	}

	// Without user read access the IDs are shown as returned.
	names, _ := c.userNamesByID(ctx)
	userName := func(id string) string {
		if n, ok := names[id]; ok {
			return n
		}
		return id
	}

	out := make([]ContainerInfo, 0, len(all))
	for _, f := range all {
		path := f.Name
		for _, p := range f.ParentInfo {
			if p.ParentType == "Project" {
				path = p.ParentName + "/" + f.Name
			}
		}
		out = append(out, ContainerInfo{
			ID:          f.ID,
			Type:        f.DocumentType,
			Name:        f.Name,
			Path:        path,
			Description: f.Description,
			CreatedTime: f.CreatedTime,
			CreatedBy:   userName(f.CreatedBy),
			UpdateTime:  f.LastUpdatedTime,
			UpdatedBy:   userName(f.LastUpdatedBy),
		})
	}
	return out, nil
}

// userNamesByID maps user IDs to user names for all users of the org.
func (c *Client) userNamesByID(ctx context.Context) (map[string]string, error) {
	names := map[string]string{}
	opts := UserListOptions{Limit: 200}
	for {
		users, err := c.ListUsers(ctx, opts)
		if err != nil {
			return nil, err
		}
		for _, u := range users {
			names[u.ID] = u.UserName
		}
		if len(users) < opts.Limit {
			return names, nil
		}
		opts.Skip += opts.Limit
	}
}

// doFRSJSON sends a GET to the FRS API on the host of the base API URL and
// decodes the JSON response. doWithSession adds the IDS-SESSION-ID header.
func (c *Client) doFRSJSON(ctx context.Context, collection string, query map[string]string, respBody interface{}) error {
	if err := c.ensureSession(ctx); err != nil {
		return err
	}
	base, err := url.Parse(c.BaseAPIURL())
	if err != nil {
		return fmt.Errorf("parsing base API URL: %w", err)
	}
	u := url.URL{Scheme: base.Scheme, Host: base.Host, Path: "/" + frsAPIPath + "/" + collection}
	q := url.Values{}
	for k, v := range query {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	resp, err := c.do(ctx, req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}
	if c.debug {
		debugPrintHTTP(req, nil, resp, respData)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return newAPIError(resp, respData)
	}
	if err := json.Unmarshal(respData, respBody); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}
	return nil
}
