package main

import (
	"fmt"
	"protoapi"
	"strings"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type protobufLinode struct {
	writer         aProtobufWriter
	instanceLabel  string
	instanceImage  string
	instanceScript string
}

func newProtobufLinode(w aProtobufWriter) *protobufLinode {
	return &protobufLinode{
		writer:         w,
		instanceLabel:  "hp_instance",
		instanceImage:  "linode/debian9",
		instanceScript: "freedom_node",
	}
}

func (p *protobufLinode) CreateTunnel(args *protoapi.LinodeCreateTunnelRequest) error {
	api := NewLinodeAPI(p.extractAuth(args.Auth))

	if err := p.ensureTunnelDoesNotExist(api, p.instanceLabel); err != nil {
		p.writer.WriteError(p.createCreateTunnelErr(err), err)
	}

	// Validate parameters.
	if len(args.Plan) == 0 {
		err := errors.New("Linode plan is empty or missing")
		return p.writer.WriteError(p.createCreateTunnelErr(err), err)
	} else if len(args.Region) == 0 {
		err := errors.New("Linode region is empty or missing")
		return p.writer.WriteError(p.createCreateTunnelErr(err), err)
	}

	// Configure builder.
	tunnelBuilder := api.NewInstanceBuilder(args.Region, args.Plan)
	tunnelBuilder.SetLabel(p.instanceLabel)
	tunnelBuilder.SetAuthorizedKeys(args.SshKeys)
	tunnelBuilder.SetImage(p.instanceImage)
	tunnelBuilder.SetBooted(true)
	tunnelBuilder.SetBackupsEnabled(false)
	tunnelBuilder.SetRootPass(args.RootPassword)

	script, params, err := p.makeStackScriptParams(
		api, p.instanceScript,
		args.RegularAccountName, args.RegularAccountPassword,
		args.WireguardOptions, args.Obfsproxy4Options, args.Obfsproxy6Options,
	)
	if err != nil {
		return p.writer.WriteError(p.createCreateTunnelErr(err), err)
	}
	tunnelBuilder.SetStackscript(script.ID, params)

	// Create instance.
	instance, err := tunnelBuilder.Create()
	if err != nil {
		p.logError(err, "Couldn't create Linode instance")
		return p.writer.WriteError(p.createCreateTunnelErr(err), err)
	}

	p.logInstance(instance, "Initiated instance creation. Waiting until it's running...")

	// Await until the instance achieves running state.
	if latest, awaitErr := p.awaitUntilRunning(api, instance.ID); awaitErr == nil {
		p.logInstance(latest, "Instance was successfully created")
		protoInstance := p.linodeInstanceToProtobuf(latest)
		return p.writer.WriteMessage(p.createCreateTunnelOK(protoInstance))
	}

	// Await returned an error, we will return old information that we've
	// received from Create().
	protoInstance := p.linodeInstanceToProtobuf(instance)
	return p.writer.WriteMessage(p.createCreateTunnelOK(protoInstance))
}

func (p *protobufLinode) RebuildTunnel(args *protoapi.LinodeRebuildTunnelRequest) error {
	api := NewLinodeAPI(p.extractAuth(args.Auth))

	tunnel, err := p.ensureTunnelExists(api, p.instanceLabel)
	if err != nil {
		return p.writer.WriteError(p.createRebuildTunnelErr(err), err)
	}

	tunnelRebuilder := api.NewInstanceRebuilder(tunnel.ID)
	tunnelRebuilder.SetAuthorizedKeys(args.SshKeys)
	tunnelRebuilder.SetBooted(true)
	tunnelRebuilder.SetImage(p.instanceImage)
	tunnelRebuilder.SetRootPass(args.RootPassword)

	script, params, err := p.makeStackScriptParams(
		api, p.instanceScript,
		args.RegularAccountName, args.RegularAccountPassword,
		args.WireguardOptions, args.Obfsproxy4Options, args.Obfsproxy6Options,
	)
	if err != nil {
		return p.writer.WriteError(p.createRebuildTunnelErr(err), err)
	}
	tunnelRebuilder.SetStackscript(script.ID, params)

	instance, err := tunnelRebuilder.Rebuild()
	if err != nil {
		p.logError(err, "Couldn't rebuild Linode instance")
		return p.writer.WriteError(p.createRebuildTunnelErr(err), err)
	}

	p.logInstance(instance, "Initiated instance rebuild. Waiting until it's running...")
	if latest, awaitErr := p.awaitUntilRunning(api, instance.ID); awaitErr == nil {
		p.logInstance(latest, "Successfully rebuilt instance")
		protoInstance := p.linodeInstanceToProtobuf(latest)
		return p.writer.WriteMessage(p.createRebuildTunnelOK(protoInstance))
	}

	// Return dated info about instance because awaitUntilRunning() has failed.
	protoInstance := p.linodeInstanceToProtobuf(instance)
	return p.writer.WriteMessage(p.createRebuildTunnelOK(protoInstance))
}

func (p *protobufLinode) DestroyTunnel(args *protoapi.LinodeDestroyTunnelRequest) error {
	api := NewLinodeAPI(p.extractAuth(args.Auth))

	tunnel, err := p.ensureTunnelExists(api, p.instanceLabel)
	if err != nil {
		return p.writer.WriteError(p.createDestroyTunnelErr(err), err)
	}

	err = api.DeleteInstance(tunnel.ID)
	if err != nil {
		p.logError(err, "Couldn't delete instance")
		return p.writer.WriteError(p.createDestroyTunnelErr(err), err)
	}
	p.logInstance(tunnel, "Instance was successfully deleted")
	return p.writer.WriteMessage(p.createDestroyTunnelOK())
}

func (p *protobufLinode) TunnelStatus(args *protoapi.LinodeGetTunnelStatusRequest) error {
	api := NewLinodeAPI(p.extractAuth(args.Auth))

	tunnel, err := p.ensureTunnelExists(api, p.instanceLabel)
	if err != nil {
		return p.writer.WriteError(p.createTunnelStatusErr(err), err)
	}
	protoTunnel := p.linodeInstanceToProtobuf(tunnel)
	return p.writer.WriteMessage(p.createTunnelStatusOK(protoTunnel))
}

func (p *protobufLinode) ListPlans(args *protoapi.LinodeListPlansRequest) error {
	plans, err := NewLinodeAPIUnauthenticated().ListInstanceTypes()
	if err != nil {
		p.logError(err, "Couldn't list Linode plans")
		return p.writer.WriteError(p.createListPlansErr(err), err)
	}

	var protoPlans []*protoapi.LinodePlan
	for _, plan := range plans {
		protoPlan := &protoapi.LinodePlan{
			Id:           plan.ID,
			Disk:         uint64(plan.Disk),
			PriceHourly:  plan.Price.Hourly,
			PriceMonthly: plan.Price.Monthly,
			Label:        plan.Label,
			NetworkOut:   uint64(plan.NetworkOut),
			Memory:       uint64(plan.Memory),
			Transfer:     uint64(plan.Transfer),
			Vcpus:        uint32(plan.VCPUs),
		}
		protoPlans = append(protoPlans, protoPlan)
	}
	return p.writer.WriteMessage(p.createListPlansOK(protoPlans))
}

func (p *protobufLinode) ListInstances(args *protoapi.LinodeListInstancesRequest) error {
	instances, err := NewLinodeAPI(p.extractAuth(args.Auth)).ListLinodeInstances()
	if err != nil {
		p.logError(err, "Couldn't list Linode instances")
		return p.writer.WriteError(p.createListInstancesErr(err), err)
	}

	var protoInstances []*protoapi.LinodeInstance
	for _, instance := range instances {
		protoInstances = append(protoInstances, p.linodeInstanceToProtobuf(&instance))
	}
	return p.writer.WriteMessage(p.createListInstancesOK(protoInstances))
}

func (p *protobufLinode) ListImages(args *protoapi.LinodeListImagesRequest) error {
	images, err := NewLinodeAPI(p.extractAuth(args.Auth)).ListLinodeImages()
	if err != nil {
		p.logError(err, "Couldn't list Linode images")
		return p.writer.WriteError(p.createListImagesErr(err), err)
	}

	var protoImages []*protoapi.LinodeImage
	for _, image := range images {
		protoImage := &protoapi.LinodeImage{
			Id:        image.ID,
			Label:     image.Label,
			Size:      uint64(image.Size),
			CreatedBy: image.CreatedBy,
			CreatedAt: image.CreatedAt,
			Vendor:    image.Vendor,
		}
		protoImages = append(protoImages, protoImage)
	}
	return p.writer.WriteMessage(p.createListImagesOK(protoImages))
}

func (p *protobufLinode) ListRegions(args *protoapi.LinodeListRegionsRequest) error {
	regions, err := NewLinodeAPIUnauthenticated().ListRegions()
	if err != nil {
		p.logError(err, "Couldn't list Linode regions")
		return p.writer.WriteError(p.createListRegionsErr(err), err)
	}

	var protoRegions []*protoapi.LinodeRegion
	for _, region := range regions {
		protoRegion := &protoapi.LinodeRegion{
			Id:      region.ID,
			Country: region.Country,
		}
		protoRegions = append(protoRegions, protoRegion)
	}
	return p.writer.WriteMessage(p.createListRegionsOK(protoRegions))
}

func (p *protobufLinode) ListStackScripts(args *protoapi.LinodeListStackScriptsRequest) error {
	scripts, err := NewLinodeAPI(p.extractAuth(args.Auth)).ListStackScriptsPrivate()
	if err != nil {
		p.logError(err, "Couldn't list Linode StackScripts")
		return p.writer.WriteError(p.createListStackScriptsErr(err), err)
	}

	var protoScripts []*protoapi.LinodeStackScript
	for _, script := range scripts {
		protoScript := &protoapi.LinodeStackScript{
			Id:          int64(script.ID),
			Label:       script.Label,
			Description: script.Description,
		}
		protoScripts = append(protoScripts, protoScript)
	}
	return p.writer.WriteMessage(p.createListStackScriptsOK(protoScripts))
}

func (p *protobufLinode) extractAuth(a *protoapi.LinodeAuth) string {
	if a != nil {
		return a.AccessToken
	}
	return ""
}

func (p *protobufLinode) awaitUntilRunning(api *LinodeAPI, instanceID int) (*LinodeInfo, error) {
	attempt, maxAttempts := 0, 20
	delay := 7 * time.Second

	// Wait a little, so operations like create or rebuild have chance to do
	// some work.
	time.Sleep(delay * 2)

	for {
		instance, err := api.QueryLinode(instanceID)
		if err != nil {
			p.logError(err, "Couldn't retrieve status of Linode instance")
			return nil, err
		}

		if instance.Status == LinodeStatusRunning {
			return instance, nil
		}

		attempt++
		if attempt >= maxAttempts {
			log.WithFields(log.Fields{
				"id":      instance.ID,
				"label":   instance.Label,
				"plan":    instance.Type,
				"ipv4":    instance.IPv4,
				"ipv6":    instance.IPv6,
				"created": instance.CreatedAt,
				"status":  instance.Status,
			}).Warn("Instance took too long to come online")
			return nil, errors.New("Instance took too long to come online")
		}
		time.Sleep(delay)
	}
}

// makeStackScriptParams produces script parameters, that are usable by either
// LinodeInstanceBuilder or LinodeInstanceRebuilder, for the instance
// initialization script.
func (p *protobufLinode) makeStackScriptParams(
	api *LinodeAPI,
	scriptName string,
	username, password string,
	wg *protoapi.WireguardOptions,
	obfs4 *protoapi.ObfsproxyIPv4Options,
	obfs6 *protoapi.ObfsproxyIPv6Options,
) (*StackScript, map[string]interface{}, error) {
	scripts, err := api.ListStackScriptsPrivate()
	if err != nil {
		p.logError(err, "Couldn't list StackScripts")
		return nil, nil, err
	}

	// Find the script by name.
	var script *StackScript
	for _, s := range scripts {
		if s.Label == scriptName {
			script = &s
		}
	}
	if script == nil {
		err = errors.New("Stackscript is missing: " + scriptName)
		p.logError(err, "Couldn't retrieve StackScript information")
		return nil, nil, err
	}

	params := make(map[string]interface{})
	params["udf_local_user_name"] = username
	params["udf_local_user_password"] = password
	if wg != nil {
		params["udf_enable_wireguard"] = 1
		params["udf_wireguard_port"] = wg.Port
		params["udf_wireguard_private_key"] = wg.ServerKey
		params["udf_wireguard_peer_keys"] = strings.Join(wg.PeerKeys, " ")
	} else {
		params["udf_enable_wireguard"] = 0
	}
	if obfs4 != nil {
		params["udf_enable_obfs4"] = 1
		params["udf_obfs4_port"] = obfs4.Port
		params["udf_obfs4_secret"] = obfs4.Secret
	} else {
		params["udf_enable_obfs4"] = 0
	}
	if obfs6 != nil {
		params["udf_enable_obfs6"] = 1
		params["udf_obfs6_port"] = obfs6.Port
		params["udf_obfs6_secret"] = obfs6.Secret
	} else {
		params["udf_enable_obfs6"] = 0
	}
	return script, params, nil
}

func (p *protobufLinode) ensureTunnelExists(api *LinodeAPI, name string) (*LinodeInfo, error) {
	tunnelInstance, err := p.retrieveTunnelInstance(api, name)
	if err != nil {
		return nil, err
	}
	if tunnelInstance == nil {
		err := errors.New("Tunnel does not exist")
		p.logError(err, "Guard failure")
		return nil, err
	}
	return tunnelInstance, nil
}

func (p *protobufLinode) ensureTunnelDoesNotExist(api *LinodeAPI, name string) error {
	tunnelInstance, err := p.retrieveTunnelInstance(api, name)
	if err != nil {
		return err
	}
	if tunnelInstance != nil {
		err := errors.New("Tunnel already exists")
		p.logError(err, "Guard failure")
		return err
	}
	return nil
}

func (p *protobufLinode) retrieveTunnelInstance(api *LinodeAPI, name string) (*LinodeInfo, error) {
	instances, err := api.ListLinodeInstances()
	if err != nil {
		p.logError(err, "Couldn't list Linode instances")
		return nil, err
	}

	// Collect all instances with matching label.
	var tunnelInstances []*LinodeInfo
	for _, instance := range instances {
		if strings.HasPrefix(instance.Label, name) {
			tunnelInstances = append(tunnelInstances, &instance)
		}
	}

	if len(tunnelInstances) >= 1 {
		if len(tunnelInstances) != 1 {
			log.
				WithField("count", len(tunnelInstances)).
				Error("Multiple tunnel instances are currently active!")
			for i, instance := range tunnelInstances {
				p.logInstance(instance, fmt.Sprintf("Active tunnel instance #%d", i))
			}
		}
		return tunnelInstances[0], nil
	}
	return nil, nil
}

func (p *protobufLinode) linodeInstanceToProtobuf(instance *LinodeInfo) *protoapi.LinodeInstance {
	status := protoapi.LinodeInstance_Status_value[strings.ToUpper(string(instance.Status))]
	return &protoapi.LinodeInstance{
		Id:         int64(instance.ID),
		Label:      instance.Label,
		Group:      instance.Group,
		Region:     instance.Region,
		Plan:       instance.Type,
		Image:      instance.Image,
		Ipv4:       instance.IPv4,
		Ipv6:       []string{instance.IPv6},
		Status:     protoapi.LinodeInstance_Status(status),
		CreatedAt:  instance.CreatedAt,
		UpdatedAt:  instance.Updated,
		Hypervisor: instance.Hypervisor,
		Disk:       uint64(instance.Specs.Disk),
		Memory:     uint64(instance.Specs.Memory),
		Vcpus:      uint32(instance.Specs.VCPUs),
		Transfer:   uint64(instance.Specs.Transfer),
	}
}

func (p *protobufLinode) logInstance(instance *LinodeInfo, msg string, extra ...log.Fields) {
	// TODO: calculate duration.
	fields := log.Fields{
		"id":         instance.ID,
		"label":      instance.Label,
		"region":     instance.Region,
		"plan":       instance.Type,
		"image":      instance.Image,
		"status":     instance.Status,
		"ipv4":       instance.IPv4,
		"ipv6":       instance.IPv6,
		"created":    instance.CreatedAt,
		"hypervisor": instance.Hypervisor,
	}

	if len(extra) > 0 {
		for k, v := range extra[0] {
			fields[k] = v
		}
	}
	log.WithFields(fields).Debug(msg)
}

func (p *protobufLinode) logError(err error, msg string) {
	log.WithFields(log.Fields{}).Error(msg)
}

func (p *protobufLinode) createError(err error) *protoapi.LinodeError {
	papiError := &protoapi.LinodeError{}
	if linodeErr, ok := err.(*LinodeError); ok {
		var errorStack []*protoapi.LinodeError_ErrorEntry
		for _, err := range linodeErr.Errors {
			entry := &protoapi.LinodeError_ErrorEntry{
				Field:  err.Field,
				Reason: err.Reason,
			}
			errorStack = append(errorStack, entry)
		}
		papiError.Details = errorStack
	} else {
		papiError.Error = &protoapi.HolepuncherError{Message: err.Error()}
	}
	return papiError
}

///////////////////////////////////////////////////////////////////////////////
// Purgatory begins here. Turn back while you can.
//

///////////////////////////////////////////////////////////////////////////////
// Responses to protoapi.LinodeCreateTunnelRequest.

func (p *protobufLinode) createCreateTunnelOK(x *protoapi.LinodeInstance) *protoapi.Response {
	return &protoapi.Response{
		R: &protoapi.Response_LinodeCreateTunnelResult{
			LinodeCreateTunnelResult: &protoapi.LinodeCreateTunnelResponse{
				Result: &protoapi.LinodeCreateTunnelResponse_Instance{Instance: x},
			},
		},
	}
}

func (p *protobufLinode) createCreateTunnelErr(err error) *protoapi.Response {
	return &protoapi.Response{
		R: &protoapi.Response_LinodeCreateTunnelResult{
			LinodeCreateTunnelResult: &protoapi.LinodeCreateTunnelResponse{
				Result: &protoapi.LinodeCreateTunnelResponse_Error{Error: p.createError(err)},
			},
		},
	}
}

///////////////////////////////////////////////////////////////////////////////
// Responses to protoapi.LinodeDestroyTunnelRequest.

func (p *protobufLinode) createDestroyTunnelOK() *protoapi.Response {
	return &protoapi.Response{
		R: &protoapi.Response_LinodeDestroyTunnelResult{
			LinodeDestroyTunnelResult: &protoapi.LinodeDestroyTunnelResponse{},
		},
	}
}

func (p *protobufLinode) createDestroyTunnelErr(err error) *protoapi.Response {
	return &protoapi.Response{
		R: &protoapi.Response_LinodeDestroyTunnelResult{
			LinodeDestroyTunnelResult: &protoapi.LinodeDestroyTunnelResponse{
				Error: p.createError(err),
			},
		},
	}
}

///////////////////////////////////////////////////////////////////////////////
// Responses to protoapi.LinodeRebuildTunnelRequest.

func (p *protobufLinode) createRebuildTunnelOK(x *protoapi.LinodeInstance) *protoapi.Response {
	return &protoapi.Response{
		R: &protoapi.Response_LinodeRebuildTunnelResult{
			LinodeRebuildTunnelResult: &protoapi.LinodeRebuildTunnelResponse{
				Result: &protoapi.LinodeRebuildTunnelResponse_Instance{Instance: x},
			},
		},
	}
}

func (p *protobufLinode) createRebuildTunnelErr(err error) *protoapi.Response {
	return &protoapi.Response{
		R: &protoapi.Response_LinodeRebuildTunnelResult{
			LinodeRebuildTunnelResult: &protoapi.LinodeRebuildTunnelResponse{
				Result: &protoapi.LinodeRebuildTunnelResponse_Error{Error: p.createError(err)},
			},
		},
	}
}

///////////////////////////////////////////////////////////////////////////////
// Responses to protoapi.LinodeGetTunnelStatusRequest.

func (p *protobufLinode) createTunnelStatusOK(x *protoapi.LinodeInstance) *protoapi.Response {
	return &protoapi.Response{
		R: &protoapi.Response_LinodeTunnelStatusResult{
			LinodeTunnelStatusResult: &protoapi.LinodeGetTunnelStatusResponse{
				Result: &protoapi.LinodeGetTunnelStatusResponse_Instance{Instance: x},
			},
		},
	}
}

func (p *protobufLinode) createTunnelStatusErr(err error) *protoapi.Response {
	return &protoapi.Response{
		R: &protoapi.Response_LinodeTunnelStatusResult{
			LinodeTunnelStatusResult: &protoapi.LinodeGetTunnelStatusResponse{
				Result: &protoapi.LinodeGetTunnelStatusResponse_Error{Error: p.createError(err)},
			},
		},
	}
}

///////////////////////////////////////////////////////////////////////////////
// Responses to protoapi.LinodeListInstancesRequest.

func (p *protobufLinode) createListInstancesOK(xs []*protoapi.LinodeInstance) *protoapi.Response {
	return &protoapi.Response{
		R: &protoapi.Response_LinodeListInstancesResult{
			LinodeListInstancesResult: &protoapi.LinodeListInstancesResponse{
				Result: &protoapi.LinodeListInstancesResponse_Instances{
					Instances: &protoapi.LinodeListInstancesResponse_List{L: xs},
				},
			},
		},
	}
}

func (p *protobufLinode) createListInstancesErr(err error) *protoapi.Response {
	return &protoapi.Response{
		R: &protoapi.Response_LinodeListInstancesResult{
			LinodeListInstancesResult: &protoapi.LinodeListInstancesResponse{
				Result: &protoapi.LinodeListInstancesResponse_Error{Error: p.createError(err)},
			},
		},
	}
}

///////////////////////////////////////////////////////////////////////////////
// Responses to protoapi.LinodeListPlansRequest.

func (p *protobufLinode) createListPlansOK(xs []*protoapi.LinodePlan) *protoapi.Response {
	return &protoapi.Response{
		R: &protoapi.Response_LinodeListPlansResult{
			LinodeListPlansResult: &protoapi.LinodeListPlansResponse{
				Result: &protoapi.LinodeListPlansResponse_Plans{
					Plans: &protoapi.LinodeListPlansResponse_List{L: xs},
				},
			},
		},
	}
}

func (p *protobufLinode) createListPlansErr(err error) *protoapi.Response {
	return &protoapi.Response{
		R: &protoapi.Response_LinodeListPlansResult{
			LinodeListPlansResult: &protoapi.LinodeListPlansResponse{
				Result: &protoapi.LinodeListPlansResponse_Error{Error: p.createError(err)},
			},
		},
	}
}

///////////////////////////////////////////////////////////////////////////////
// Responses to protoapi.LinodeListImagesRequest.

func (p *protobufLinode) createListImagesOK(xs []*protoapi.LinodeImage) *protoapi.Response {
	return &protoapi.Response{
		R: &protoapi.Response_LinodeListImagesResult{
			LinodeListImagesResult: &protoapi.LinodeListImagesResponse{
				Result: &protoapi.LinodeListImagesResponse_Images{
					Images: &protoapi.LinodeListImagesResponse_List{L: xs},
				},
			},
		},
	}
}

func (p *protobufLinode) createListImagesErr(err error) *protoapi.Response {
	return &protoapi.Response{
		R: &protoapi.Response_LinodeListImagesResult{
			LinodeListImagesResult: &protoapi.LinodeListImagesResponse{
				Result: &protoapi.LinodeListImagesResponse_Error{Error: p.createError(err)},
			},
		},
	}
}

///////////////////////////////////////////////////////////////////////////////
// Responses to protoapi.LinodeListRegionsRequest.

func (p *protobufLinode) createListRegionsOK(xs []*protoapi.LinodeRegion) *protoapi.Response {
	return &protoapi.Response{
		R: &protoapi.Response_LinodeListRegionsResult{
			LinodeListRegionsResult: &protoapi.LinodeListRegionsResponse{
				Result: &protoapi.LinodeListRegionsResponse_Regions{
					Regions: &protoapi.LinodeListRegionsResponse_List{L: xs},
				},
			},
		},
	}
}

func (p *protobufLinode) createListRegionsErr(err error) *protoapi.Response {
	return &protoapi.Response{
		R: &protoapi.Response_LinodeListRegionsResult{
			LinodeListRegionsResult: &protoapi.LinodeListRegionsResponse{
				Result: &protoapi.LinodeListRegionsResponse_Error{Error: p.createError(err)},
			},
		},
	}
}

///////////////////////////////////////////////////////////////////////////////
// Responses to protoapi.LinodeListStackScriptsRequest.

func (p *protobufLinode) createListStackScriptsOK(xs []*protoapi.LinodeStackScript) *protoapi.Response {
	return &protoapi.Response{
		R: &protoapi.Response_LinodeListStackscriptsResult{
			LinodeListStackscriptsResult: &protoapi.LinodeListStackScriptsResponse{
				Result: &protoapi.LinodeListStackScriptsResponse_Stackscripts{
					Stackscripts: &protoapi.LinodeListStackScriptsResponse_List{L: xs},
				},
			},
		},
	}
}

func (p *protobufLinode) createListStackScriptsErr(err error) *protoapi.Response {
	return &protoapi.Response{
		R: &protoapi.Response_LinodeListStackscriptsResult{
			LinodeListStackscriptsResult: &protoapi.LinodeListStackScriptsResponse{
				Result: &protoapi.LinodeListStackScriptsResponse_Error{Error: p.createError(err)},
			},
		},
	}
}
