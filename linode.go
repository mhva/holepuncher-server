package main

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
	resty "gopkg.in/resty.v1"
)

// LinodeAPI is an entry-point type through which all interactions with
// Linode API are performed.
type LinodeAPI struct {
	apiKey string
	client *resty.Client
}

// LinodeError represents a Linode error.
type LinodeError struct {
	Errors []struct {
		Field  string `json:"field"`
		Reason string `json:"reason"`
	} `json:"errors"`

	isAuthError        bool
	isPermissionsError bool
}

// LinodeInfo contains a description of a single active Linode instance.
type LinodeInfo struct {
	ID         int          `json:"id"`
	Region     string       `json:"region"`
	Image      string       `json:"image"`
	IPv4       []string     `json:"ipv4"`
	IPv6       string       `json:"ipv6"`
	Label      string       `json:"label"`
	Group      string       `json:"group"`
	Type       string       `json:"type"`
	Status     LinodeStatus `json:"status"`
	CreatedAt  string       `json:"created"`
	Updated    string       `json:"updated"`
	Hypervisor string       `json:"hypervisor"`
	Specs      struct {
		Disk     int `json:"disk"`
		Memory   int `json:"memory"`
		VCPUs    int `json:"vcpus"`
		Transfer int `json:"transfer"`
	} `json:"specs"`
}

// StackScript is a struct containing a single StackScript description.
type StackScript struct {
	ID          int      `json:"id"`
	Label       string   `json:"label"`
	Description string   `json:"description"`
	Images      []string `json:"images"`
	IsPublic    bool     `json:"is_public"`
}

// LinodeRegion is a struct containing a single Linode region description.
type LinodeRegion struct {
	ID      string `json:"id"`
	Country string `json:"country"`
}

// LinodeImage is a struct containing a description of single deployable
// Linode image.
type LinodeImage struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description"`
	IsPublic    bool   `json:"is_public"`
	Size        int    `json:"size"`
	CreatedBy   string `json:"created_by"`
	CreatedAt   string `json:"created"`
	Vendor      string `json:"vendor"`
	Deprecated  bool   `json:"deprecated"`
}

// LinodeType is a struct containing a single Linode type description.
type LinodeType struct {
	ID         string `json:"id"`
	Disk       int    `json:"disk"`
	Label      string `json:"label"`
	NetworkOut int    `json:"network_out"`
	Memory     int    `json:"memory"`
	Transfer   int    `json:"transfer"`
	VCPUs      int    `json:"vcpus"`
	Price      struct {
		Hourly  float32 `json:"hourly"`
		Monthly float32 `json:"monthly"`
	} `json:"price"`
}

// LinodeInstanceBuilder provides a comprehensive set of methods for configuring
// new Linode instance.
type LinodeInstanceBuilder struct {
	api             *LinodeAPI
	Region          string                 `json:"region"`
	Type            string                 `json:"type"`
	Label           string                 `json:"label,omitempty"`
	Group           string                 `json:"group,omitempty"`
	RootPass        string                 `json:"root_pass,omitempty"`
	AuthorizedKeys  []string               `json:"authorized_keys,omitempty"`
	StackscriptID   int                    `json:"stackscript_id,omitempty"`
	StackscriptData map[string]interface{} `json:"stackscript_data,omitempty"`
	BackupID        int                    `json:"backup_id,omitempty"`
	Image           string                 `json:"image,omitempty"`
	BackupsEnabled  bool                   `json:"backups_enabled,omitempty"`
	Booted          bool                   `json:"booted,omitempty"`
}

// LinodeInstanceRebuilder provides a way to rebuild existing Linode instance.
type LinodeInstanceRebuilder struct {
	api             *LinodeAPI
	id              int
	RootPass        string                 `json:"root_pass,omitempty"`
	AuthorizedKeys  []string               `json:"authorized_keys,omitempty"`
	StackscriptID   int                    `json:"stackscript_id,omitempty"`
	StackscriptData map[string]interface{} `json:"stackscript_data,omitempty"`
	Image           string                 `json:"image,omitempty"`
	Booted          bool                   `json:"booted,omitempty"`
}

// LinodeStatus enum describes status of an active Linode.
type LinodeStatus string

const (
	// LinodeStatusOffline indicates that Linode is offline.
	LinodeStatusOffline LinodeStatus = "offline"
	// LinodeStatusBooting indicates that Linode is booting.
	LinodeStatusBooting LinodeStatus = "booting"
	// LinodeStatusRunning indicates that Linode is running.
	LinodeStatusRunning LinodeStatus = "running"
	// LinodeStatusShuttingDown indicates that Linode is shutting down.
	LinodeStatusShuttingDown LinodeStatus = "shutting_down"
	// LinodeStatusRebooting indicates that Linode is rebooting.
	LinodeStatusRebooting LinodeStatus = "rebooting"
	// LinodeStatusProvisioning indicates that Linode is being provisioned.
	LinodeStatusProvisioning LinodeStatus = "provisioning"
	// LinodeStatusDeleting indicates that Linode is being deleted.
	LinodeStatusDeleting LinodeStatus = "deleting"
	// LinodeStatusMigrating indicates that Linode is being migrated.
	LinodeStatusMigrating LinodeStatus = "migrating"
	// LinodeStatusRestoring indicates that Linode is being restored from backup.
	LinodeStatusRestoring LinodeStatus = "restoring"
	// LinodeStatusRebuilding indicates that Linode is being rebuilt.
	LinodeStatusRebuilding LinodeStatus = "rebuilding"
	// LinodeStatusCloning indicates that Linode is being cloned.
	LinodeStatusCloning LinodeStatus = "cloning"
)

// NewLinodeAPI creates an authenticated LinodeAPI instance that can be used
// to access any API endpoint without restrictions (assuming you have appropriate
// access permissions).
func NewLinodeAPI(apiKey string) *LinodeAPI {
	client := resty.New()
	client.SetAuthToken(apiKey)
	client.SetError(&LinodeError{})
	client.SetTimeout(60 * time.Second)
	client.SetHeader("User-Agent", "linode_client")

	client.SetDebug(true)

	return &LinodeAPI{
		apiKey: apiKey,
		client: client,
	}
}

// NewLinodeAPIUnauthenticated creates an unauthenticated LinodeAPI instance that
// has access to API endpoints that do not require authentication.
func NewLinodeAPIUnauthenticated() *LinodeAPI {
	client := resty.New()
	client.SetError(&LinodeError{})
	client.SetTimeout(60 * time.Second)
	client.SetHeader("User-Agent", "linode_client")

	client.SetDebug(true)

	return &LinodeAPI{
		client: client,
	}
}

// NewInstanceBuilder creates a LinodeInstanceBuilder used to create a new
// Linode instance.
func (e *LinodeAPI) NewInstanceBuilder(region string, linodeType string) *LinodeInstanceBuilder {
	return &LinodeInstanceBuilder{
		api:    e,
		Region: region,
		Type:   linodeType,
	}
}

// NewInstanceRebuilder creates a LinodeInstanceRebuilder used to rebuild an
// existing Linode instance.
func (e *LinodeAPI) NewInstanceRebuilder(id int) *LinodeInstanceRebuilder {
	return &LinodeInstanceRebuilder{
		api: e,
		id:  id,
	}
}

// BootInstance attempts to boot specified instance.
func (e *LinodeAPI) BootInstance(linodeID int) error {
	var dummy map[string]interface{}
	endpoint := fmt.Sprintf("/linode/instances/%d/boot", linodeID)
	result := linodePOST(endpoint, e.authedR().SetResult(&dummy))

	if result.err == nil {
		return nil
	}
	return errors.Wrapf(result.err, "Unable to boot instance")
}

// DeleteInstance irreversibly deletes an existing instance.
func (e *LinodeAPI) DeleteInstance(linodeID int) error {
	var dummy map[string]interface{}

	endpoint := fmt.Sprintf("/linode/instances/%d", linodeID)
	client := e.authedR().SetResult(&dummy)
	result := linodeDELETE(endpoint, client)

	if result.err == nil {
		return nil
	}
	return errors.Wrapf(result.err, "Unable to delete instance")
}

// QueryLinode returns information about a linode.
func (e *LinodeAPI) QueryLinode(linodeID int) (*LinodeInfo, error) {
	endpoint := fmt.Sprintf("/linode/instances/%d", linodeID)
	r := e.authedR().SetResult(&LinodeInfo{})
	result := linodeGET(endpoint, r)

	if result.err != nil {
		return nil, result.err
	}

	if linodeInfo, ok := result.data.(*LinodeInfo); ok {
		return linodeInfo, nil
	}
	return nil, errors.New("unable to decode RPC return value (" + endpoint + ")")
}

// ListLinodeInstances returns a list of active linodes.
func (e *LinodeAPI) ListLinodeInstances() ([]LinodeInfo, error) {
	endpoint := "/linode/instances"
	r := e.authedR().SetResult([]LinodeInfo{})
	iter := linodePaginatedGET(endpoint, r, &linodeInfoPaginated{})
	list := []LinodeInfo{}

	for {
		item, hasNext := iter.next()
		if item.err != nil {
			return list, item.err
		}
		if moreItems, ok := item.data.([]LinodeInfo); ok {
			list = append(list, moreItems...)
		} else {
			err := errors.New("unable to decode RPC return value (" + endpoint + ")")
			return list, err
		}
		if !hasNext {
			break
		}
	}
	return list, nil
}

// ListStackScriptsPrivate returns a list of all private StackScripts.
func (e *LinodeAPI) ListStackScriptsPrivate() ([]StackScript, error) {
	endpoint := "/linode/stackscripts"
	r := e.authedR().SetResult([]StackScript{}).SetHeader("X-Filter", `{"mine": true}`)
	iter := linodePaginatedGET(endpoint, r, &stackScriptPaginated{})
	list := []StackScript{}

	for {
		item, hasNext := iter.next()
		if item.err != nil {
			return list, item.err
		}
		if moreItems, ok := item.data.([]StackScript); ok {
			list = append(list, moreItems...)
		} else {
			err := errors.New("unable to decode RPC return value (" + endpoint + ")")
			return list, err
		}
		if !hasNext {
			break
		}
	}
	return list, nil
}

// ListLinodeImages returns a list of deployable images.
func (e *LinodeAPI) ListLinodeImages() ([]LinodeImage, error) {
	endpoint := "/images"
	r := e.authedR().SetResult([]LinodeImage{})
	iter := linodePaginatedGET(endpoint, r, &linodeImagePaginated{})
	list := []LinodeImage{}

	for {
		item, hasNext := iter.next()
		if item.err != nil {
			return list, item.err
		}
		if moreItems, ok := item.data.([]LinodeImage); ok {
			list = append(list, moreItems...)
		} else {
			err := errors.New("unable to decode RPC return value (" + endpoint + ")")
			return list, err
		}
		if !hasNext {
			break
		}
	}
	return list, nil
}

// ListInstanceTypes returns a list of supported instance types.
// Can be used without authentication.
func (e *LinodeAPI) ListInstanceTypes() ([]LinodeType, error) {
	endpoint := "/linode/types"
	r := e.unprivR().SetResult([]LinodeType{})
	iter := linodePaginatedGET(endpoint, r, &linodeTypePaginated{})
	list := []LinodeType{}

	for {
		item, hasNext := iter.next()
		if item.err != nil {
			return list, item.err
		}
		if moreItems, ok := item.data.([]LinodeType); ok {
			list = append(list, moreItems...)
		} else {
			err := errors.New("unable to decode RPC return value (" + endpoint + ")")
			return list, err
		}
		if !hasNext {
			break
		}
	}
	return list, nil
}

// ListRegions returns a list of supported geographic regions.
// Can be used without authentication.
func (e *LinodeAPI) ListRegions() ([]LinodeRegion, error) {
	endpoint := "/regions"
	r := e.unprivR().SetResult([]LinodeRegion{})
	iter := linodePaginatedGET(endpoint, r, &linodeRegionPaginated{})
	list := []LinodeRegion{}

	for {
		item, hasNext := iter.next()
		if item.err != nil {
			return list, item.err
		}
		if moreItems, ok := item.data.([]LinodeRegion); ok {
			list = append(list, moreItems...)
		} else {
			err := errors.New("unable to decode RPC return value (" + endpoint + ")")
			return list, err
		}
		if !hasNext {
			break
		}
	}
	return list, nil
}

func (e *LinodeError) Error() string {
	var result string
	for n, err := range e.Errors {
		if len(err.Field) > 0 {
			result += fmt.Sprintf("%s (field '%s')", err.Reason, err.Field)
		} else {
			result += err.Reason
		}
		if n+1 < len(e.Errors) {
			result += ";"
		}
	}

	return result
}

// IsAuthError checks whether the error is an authorization error.
func (e *LinodeError) IsAuthError() bool {
	return e.isAuthError
}

// IsPermissionsError return true when error was caused by insufficient
// user permissions.
func (e *LinodeError) IsPermissionsError() bool {
	return e.isPermissionsError
}

func (e *LinodeAPI) unprivR() *resty.Request {
	return e.client.R().SetError(&LinodeError{})
}

func (e *LinodeAPI) authedR() *resty.Request {
	if len(e.apiKey) > 0 {
		return e.client.R().SetError(&LinodeError{})
	}
	panic("Attempted to perform authenticated request, but this LinodeAPI instance has no API key")
}

// SetLabel sets Linode label.
func (e *LinodeInstanceBuilder) SetLabel(label string) *LinodeInstanceBuilder {
	e.Label = label
	return e
}

// SetGroup sets Linode group name.
func (e *LinodeInstanceBuilder) SetGroup(group string) *LinodeInstanceBuilder {
	e.Group = group
	return e
}

// SetRootPass sets Linode password. This setting must be set, if image is provided.
func (e *LinodeInstanceBuilder) SetRootPass(rootPass string) *LinodeInstanceBuilder {
	e.RootPass = rootPass
	return e
}

// SetAuthorizedKeys sets a list of authorized SSH keys for default user.
func (e *LinodeInstanceBuilder) SetAuthorizedKeys(keys []string) *LinodeInstanceBuilder {
	e.AuthorizedKeys = keys
	return e
}

// SetStackscript sets StackScript that will be used during provisioning.
func (e *LinodeInstanceBuilder) SetStackscript(
	id int,
	data ...map[string]interface{},
) *LinodeInstanceBuilder {
	e.StackscriptID = id
	if len(data) > 0 {
		e.StackscriptData = data[0]
	}
	return e
}

// SetBackupID sets backup an existing backup image as a source for creating
// new Linode.
func (e *LinodeInstanceBuilder) SetBackupID(backupID int) *LinodeInstanceBuilder {
	e.BackupID = backupID
	return e
}

// SetImage sets a premade image as a source for creating new Linode.
func (e *LinodeInstanceBuilder) SetImage(image string) *LinodeInstanceBuilder {
	e.Image = image
	return e
}

// SetBackupsEnabled enables backup function for new Linode.
func (e *LinodeInstanceBuilder) SetBackupsEnabled(enabled bool) *LinodeInstanceBuilder {
	e.BackupsEnabled = enabled
	return e
}

// SetBooted controls whether Linode should be automatically booted after
// creation.
func (e *LinodeInstanceBuilder) SetBooted(booted bool) *LinodeInstanceBuilder {
	e.Booted = booted
	return e
}

// Create finalizes current builder and creates new Linode!
func (e *LinodeInstanceBuilder) Create() (*LinodeInfo, error) {
	endpoint := "/linode/instances"
	r := e.api.authedR().SetBody(e).SetResult(&LinodeInfo{})
	result := linodePOST(endpoint, r)

	if result.err != nil {
		return nil, result.err
	}

	if instance, ok := result.response.Result().(*LinodeInfo); ok {
		return instance, nil
	}
	return nil, errors.New("unable to parse RPC result")
}

// SetRootPass sets Linode password. This setting must be set, if image is provided.
func (r *LinodeInstanceRebuilder) SetRootPass(rootPass string) *LinodeInstanceRebuilder {
	r.RootPass = rootPass
	return r
}

// SetAuthorizedKeys sets a list of authorized SSH keys for default user.
func (r *LinodeInstanceRebuilder) SetAuthorizedKeys(keys []string) *LinodeInstanceRebuilder {
	r.AuthorizedKeys = keys
	return r
}

// SetStackscript sets StackScript that will be used during provisioning.
func (r *LinodeInstanceRebuilder) SetStackscript(
	id int,
	data ...map[string]interface{},
) *LinodeInstanceRebuilder {
	r.StackscriptID = id
	if len(data) > 0 {
		r.StackscriptData = data[0]
	}
	return r
}

// SetImage sets a premade image as a source for creating new Linode.
func (r *LinodeInstanceRebuilder) SetImage(image string) *LinodeInstanceRebuilder {
	r.Image = image
	return r
}

// SetBooted controls whether Linode should be automatically booted after
// rebuilding.
func (r *LinodeInstanceRebuilder) SetBooted(booted bool) *LinodeInstanceRebuilder {
	r.Booted = booted
	return r
}

// Rebuild rebuilds a Linode.
func (r *LinodeInstanceRebuilder) Rebuild() (*LinodeInfo, error) {
	endpoint := fmt.Sprintf("/linode/instances/%d/rebuild", r.id)
	client := r.api.authedR().SetBody(r).SetResult(&LinodeInfo{})
	result := linodePOST(endpoint, client)

	if result.err != nil {
		return nil, result.err
	}

	if instance, ok := result.response.Result().(*LinodeInfo); ok {
		return instance, nil
	}
	return nil, errors.New("unable to parse RPC result")
}
