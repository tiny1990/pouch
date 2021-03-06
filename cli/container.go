package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/alibaba/pouch/apis/types"
	"github.com/alibaba/pouch/pkg/runconfig"
	"github.com/docker/go-connections/nat"

	units "github.com/docker/go-units"
	strfmt "github.com/go-openapi/strfmt"
)

type container struct {
	labels           []string
	name             string
	tty              bool
	volume           []string
	runtime          string
	env              []string
	entrypoint       string
	workdir          string
	user             string
	groupAdd         []string
	hostname         string
	cpushare         int64
	cpusetcpus       string
	cpusetmems       string
	memory           string
	memorySwap       string
	memorySwappiness int64

	memoryWmarkRatio    int64
	memoryExtra         int64
	memoryForceEmptyCtl int64
	scheLatSwitch       int64
	oomKillDisable      bool

	devices              []string
	enableLxcfs          bool
	privileged           bool
	restartPolicy        string
	ipcMode              string
	pidMode              string
	utsMode              string
	sysctls              []string
	networks             []string
	ports                []string
	expose               []string
	publicAll            bool
	securityOpt          []string
	capAdd               []string
	capDrop              []string
	blkioWeight          uint16
	blkioWeightDevice    WeightDevice
	blkioDeviceReadBps   ThrottleBpsDevice
	blkioDeviceWriteBps  ThrottleBpsDevice
	blkioDeviceReadIOps  ThrottleIOpsDevice
	blkioDeviceWriteIOps ThrottleIOpsDevice
	IntelRdtL3Cbm        string
	diskQuota            []string
	oomScoreAdj          int64

	cgroupParent string

	//add for rich container mode
	rich       bool
	richMode   string
	initScript string
}

func (c *container) config() (*types.ContainerCreateConfig, error) {
	labels, err := parseLabels(c.labels)
	if err != nil {
		return nil, err
	}

	if err := validateMemorySwappiness(c.memorySwappiness); err != nil {
		return nil, err
	}

	memory, err := parseMemory(c.memory)
	if err != nil {
		return nil, err
	}

	memorySwap, err := parseMemorySwap(c.memorySwap)
	if err != nil {
		return nil, err
	}

	intelRdtL3Cbm, err := parseIntelRdt(c.IntelRdtL3Cbm)
	if err != nil {
		return nil, err
	}

	deviceMappings, err := parseDeviceMappings(c.devices)
	if err != nil {
		return nil, err
	}

	restartPolicy, err := parseRestartPolicy(c.restartPolicy)
	if err != nil {
		return nil, err
	}

	sysctls, err := parseSysctls(c.sysctls)
	if err != nil {
		return nil, err
	}

	diskQuota, err := parseDiskQuota(c.diskQuota)
	if err != nil {
		return nil, err
	}

	if err := validateOOMScore(c.oomScoreAdj); err != nil {
		return nil, err
	}

	var networkMode string
	if len(c.networks) == 0 {
		networkMode = "bridge"
	}
	networkingConfig := &types.NetworkingConfig{
		EndpointsConfig: map[string]*types.EndpointSettings{},
	}
	for _, network := range c.networks {
		name, parameter, mode, err := parseNetwork(network)
		if err != nil {
			return nil, err
		}

		if networkMode == "" || mode == "mode" {
			networkMode = name
		}

		if name == "container" {
			networkMode = fmt.Sprintf("%s:%s", name, parameter)
		} else if ipaddr := net.ParseIP(parameter); ipaddr != nil {
			networkingConfig.EndpointsConfig[name] = &types.EndpointSettings{
				IPAddress: parameter,
				IPAMConfig: &types.EndpointIPAMConfig{
					IPV4Address: parameter,
				},
			}
		}
	}

	// parse port binding
	tmpPorts, tmpPortBindings, err := nat.ParsePortSpecs(c.ports)
	if err != nil {
		return nil, err
	}
	// translate ports and portbingings
	ports := map[string]interface{}{}
	for n, p := range tmpPorts {
		ports[string(n)] = p
	}
	portBindings := make(types.PortMap)
	for n, pbs := range tmpPortBindings {
		portBindings[string(n)] = []types.PortBinding{}
		for _, tmpPb := range pbs {
			pb := types.PortBinding{HostIP: tmpPb.HostIP, HostPort: tmpPb.HostPort}
			portBindings[string(n)] = append(portBindings[string(n)], pb)
		}
	}

	for _, e := range c.expose {
		if strings.Contains(e, ":") {
			return nil, fmt.Errorf("invalid port format for --expose: %s", e)
		}

		//support two formats for expose, original format <portnum>/[<proto>] or <startport-endport>/[<proto>]
		proto, port := nat.SplitProtoPort(e)
		//parse the start and end port and create a sequence of ports to expose
		//if expose a port, the start and end port are the same
		start, end, err := nat.ParsePortRange(port)
		if err != nil {
			return nil, fmt.Errorf("invalid range format for --expose: %s, error: %s", e, err)
		}
		for i := start; i <= end; i++ {
			p, err := nat.NewPort(proto, strconv.FormatUint(i, 10))
			if err != nil {
				return nil, err
			}
			if _, exists := ports[string(p)]; !exists {
				ports[string(p)] = struct{}{}
			}
		}
	}
	config := &types.ContainerCreateConfig{
		ContainerConfig: types.ContainerConfig{
			Tty:          c.tty,
			Env:          c.env,
			Entrypoint:   strings.Fields(c.entrypoint),
			WorkingDir:   c.workdir,
			User:         c.user,
			Hostname:     strfmt.Hostname(c.hostname),
			Labels:       labels,
			Rich:         c.rich,
			RichMode:     c.richMode,
			InitScript:   c.initScript,
			ExposedPorts: ports,
			DiskQuota:    diskQuota,
		},

		HostConfig: &types.HostConfig{
			Binds:   c.volume,
			Runtime: c.runtime,
			Resources: types.Resources{
				CPUShares:        c.cpushare,
				CpusetCpus:       c.cpusetcpus,
				CpusetMems:       c.cpusetmems,
				Devices:          deviceMappings,
				Memory:           memory,
				MemorySwap:       memorySwap,
				MemorySwappiness: &c.memorySwappiness,
				// FIXME: validate in client side
				MemoryWmarkRatio:    &c.memoryWmarkRatio,
				MemoryExtra:         &c.memoryExtra,
				MemoryForceEmptyCtl: c.memoryForceEmptyCtl,
				ScheLatSwitch:       c.scheLatSwitch,
				OomKillDisable:      &c.oomKillDisable,

				// blkio
				BlkioWeight:          c.blkioWeight,
				BlkioWeightDevice:    c.blkioWeightDevice.value(),
				BlkioDeviceReadBps:   c.blkioDeviceReadBps.value(),
				BlkioDeviceReadIOps:  c.blkioDeviceReadIOps.value(),
				BlkioDeviceWriteBps:  c.blkioDeviceWriteBps.value(),
				BlkioDeviceWriteIOps: c.blkioDeviceWriteIOps.value(),
				IntelRdtL3Cbm:        intelRdtL3Cbm,

				CgroupParent: c.cgroupParent,
			},
			EnableLxcfs:   c.enableLxcfs,
			Privileged:    c.privileged,
			RestartPolicy: restartPolicy,
			IpcMode:       c.ipcMode,
			PidMode:       c.pidMode,
			UTSMode:       c.utsMode,
			GroupAdd:      c.groupAdd,
			Sysctls:       sysctls,
			SecurityOpt:   c.securityOpt,
			NetworkMode:   networkMode,
			CapAdd:        c.capAdd,
			CapDrop:       c.capDrop,
			PortBindings:  portBindings,
			OomScoreAdj:   c.oomScoreAdj,
		},

		NetworkingConfig: networkingConfig,
	}

	return config, nil
}

func parseSysctls(sysctls []string) (map[string]string, error) {
	results := make(map[string]string)
	for _, sysctl := range sysctls {
		fields, err := parseSysctl(sysctl)
		if err != nil {
			return nil, err
		}
		k, v := fields[0], fields[1]
		results[k] = v
	}
	return results, nil
}

func parseSysctl(sysctl string) ([]string, error) {
	fields := strings.SplitN(sysctl, "=", 2)
	if len(fields) != 2 {
		return nil, fmt.Errorf("invalid sysctl %s: sysctl must be in format of key=value", sysctl)
	}
	return fields, nil
}

func parseLabels(labels []string) (map[string]string, error) {
	results := make(map[string]string)
	for _, label := range labels {
		fields, err := parseLabel(label)
		if err != nil {
			return nil, err
		}
		k, v := fields[0], fields[1]
		results[k] = v
	}
	return results, nil
}

func parseLabel(label string) ([]string, error) {
	fields := strings.SplitN(label, "=", 2)
	if len(fields) != 2 {
		return nil, fmt.Errorf("invalid label %s: label must be in format of key=value", label)
	}
	return fields, nil
}

func parseDeviceMappings(devices []string) ([]*types.DeviceMapping, error) {
	results := []*types.DeviceMapping{}
	for _, device := range devices {
		deviceMapping, err := parseDevice(device)
		if err != nil {
			return nil, err
		}
		results = append(results, deviceMapping)
	}
	return results, nil
}

func parseDevice(device string) (*types.DeviceMapping, error) {
	deviceMapping, err := runconfig.ParseDevice(device)
	if err != nil {
		return nil, fmt.Errorf("parse devices error: %s", err)
	}
	if !runconfig.ValidDeviceMode(deviceMapping.CgroupPermissions) {
		return nil, fmt.Errorf("%s invalid device mode: %s", device, deviceMapping.CgroupPermissions)
	}
	return deviceMapping, nil
}

func parseMemory(memory string) (int64, error) {
	if memory == "" {
		return 0, nil
	}
	result, err := units.RAMInBytes(memory)
	if err != nil {
		return 0, err
	}
	return result, nil
}

func parseMemorySwap(memorySwap string) (int64, error) {
	if memorySwap == "" {
		return 0, nil
	}
	if memorySwap == "-1" {
		return -1, nil
	}
	result, err := units.RAMInBytes(memorySwap)
	if err != nil {
		return 0, err
	}
	return result, nil
}

func validateMemorySwappiness(memorySwappiness int64) error {
	if memorySwappiness != -1 && (memorySwappiness < 0 || memorySwappiness > 100) {
		return fmt.Errorf("invalid memory swappiness: %d (its range is -1 or 0-100)", memorySwappiness)
	}
	return nil
}

func parseIntelRdt(intelRdtL3Cbm string) (string, error) {
	// FIXME: add Intel RDT L3 Cbm validation
	return intelRdtL3Cbm, nil
}

func parseRestartPolicy(restartPolicy string) (*types.RestartPolicy, error) {
	policy := &types.RestartPolicy{}

	if restartPolicy == "" {
		policy.Name = "no"
		return policy, nil
	}

	fields := strings.Split(restartPolicy, ":")
	policy.Name = fields[0]

	switch policy.Name {
	case "always", "unless-stopped", "no":
	case "on-failure":
		if len(fields) > 2 {
			return nil, fmt.Errorf("invalid restart policy: %s", restartPolicy)
		}
		if len(fields) == 2 {
			n, err := strconv.Atoi(fields[1])
			if err != nil {
				return nil, fmt.Errorf("invalid restart policy: %v", err)
			}
			policy.MaximumRetryCount = int64(n)
		}
	default:
		return nil, fmt.Errorf("invalid restart policy: %s", restartPolicy)
	}

	return policy, nil
}

// network format as below:
// [network]:[ip_address], such as: mynetwork:172.17.0.2 or mynetwork(ip alloc by ipam) or 172.17.0.2(default network is bridge)
// [network_mode]:[parameter], such as: host(use host network) or container:containerID(use exist container network)
// [network_mode]:[parameter]:mode, such as: mynetwork:172.17.0.2:mode(if the container has multi-networks, the network is the default network mode)
func parseNetwork(network string) (string, string, string, error) {
	var (
		name      string
		parameter string
		mode      string
	)
	if network == "" {
		return "", "", "", fmt.Errorf("invalid network: cannot be empty")
	}
	arr := strings.Split(network, ":")
	switch len(arr) {
	case 1:
		if ipaddr := net.ParseIP(arr[0]); ipaddr != nil {
			parameter = arr[0]
		} else {
			name = arr[0]
		}
	case 2:
		name = arr[0]
		if name == "container" {
			parameter = arr[1]
		} else if ipaddr := net.ParseIP(arr[1]); ipaddr != nil {
			parameter = arr[1]
		} else {
			mode = arr[1]
		}
	default:
		name = arr[0]
		parameter = arr[1]
		mode = arr[2]
	}

	return name, parameter, mode, nil
}

func parseDiskQuota(quotas []string) (map[string]string, error) {
	var quotaMaps = make(map[string]string)

	for _, quota := range quotas {
		if quota == "" {
			return nil, fmt.Errorf("invalid format for disk quota: %s", quota)
		}

		parts := strings.Split(quota, "=")
		switch len(parts) {
		case 1:
			quotaMaps["/"] = parts[0]
		case 2:
			quotaMaps[parts[0]] = parts[1]
		default:
			return nil, fmt.Errorf("invalid format for disk quota: %s", quota)
		}
	}

	return quotaMaps, nil
}

// validateOOMScore validates oom score
func validateOOMScore(score int64) error {
	if score < -1000 || score > 1000 {
		return fmt.Errorf("oom-score-adj should be in range [-1000, 1000]")
	}

	return nil
}
