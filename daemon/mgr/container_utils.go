package mgr

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/alibaba/pouch/apis/types"
	"github.com/alibaba/pouch/pkg/errtypes"
	"github.com/alibaba/pouch/pkg/meta"
	"github.com/alibaba/pouch/pkg/randomid"

	"github.com/pkg/errors"
)

// containerID returns the container's id, the parameter 'nameOrPrefix' may be container's
// name, id or prefix id.
func (mgr *ContainerManager) containerID(nameOrPrefix string) (string, error) {
	var obj meta.Object

	// name is the container's name.
	id, ok := mgr.NameToID.Get(nameOrPrefix).String()
	if ok {
		return id, nil
	}

	// name is the container's prefix of the id.
	objs, err := mgr.Store.GetWithPrefix(nameOrPrefix)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get container info with prefix: %s", nameOrPrefix)
	}
	if len(objs) > 1 {
		return "", errors.Wrap(errtypes.ErrTooMany, "container: "+nameOrPrefix)
	}
	if len(objs) == 0 {
		return "", errors.Wrap(errtypes.ErrNotfound, "container: "+nameOrPrefix)
	}
	obj = objs[0]

	containerMeta, ok := obj.(*ContainerMeta)
	if !ok {
		return "", fmt.Errorf("failed to get container info, invalid meta's type")
	}

	return containerMeta.ID, nil
}

func (mgr *ContainerManager) container(nameOrPrefix string) (*Container, error) {
	res, ok := mgr.cache.Get(nameOrPrefix).Result()
	if ok {
		return res.(*Container), nil
	}

	id, err := mgr.containerID(nameOrPrefix)
	if err != nil {
		return nil, err
	}

	// lookup again
	res, ok = mgr.cache.Get(id).Result()
	if ok {
		return res.(*Container), nil
	}

	return nil, errors.Wrap(errtypes.ErrNotfound, "container "+nameOrPrefix)
}

// generateID generates an ID for newly created container. We must ensure that
// this ID has not used yet.
func (mgr *ContainerManager) generateID() (string, error) {
	var id string
	for {
		id = randomid.Generate()
		_, err := mgr.Store.Get(id)
		if err != nil {
			if merr, ok := err.(meta.Error); ok && merr.IsNotfound() {
				break
			}
			return "", err
		}
	}
	return id, nil
}

// generateName generates container name by container ID.
// It get first 6 continuous letters which has not been taken.
// TODO: take a shorter than 6 letters ID into consideration.
// FIXME: there is possibility that for block loops forever.
func (mgr *ContainerManager) generateName(id string) string {
	var name string
	i := 0
	for {
		if i+6 > len(id) {
			break
		}
		name = id[i : i+6]
		i++
		if !mgr.NameToID.Get(name).Exist() {
			break
		}
	}
	return name
}

func parseSecurityOpts(meta *ContainerMeta, securityOpts []string) error {
	for _, securityOpt := range securityOpts {
		if err := parseSecurityOpt(meta, securityOpt); err != nil {
			return err
		}
	}
	return nil
}

func parseSecurityOpt(meta *ContainerMeta, securityOpt string) error {
	fields := strings.SplitN(securityOpt, "=", 2)
	if len(fields) != 2 {
		return fmt.Errorf("invalid --security-opt %s: must be in format of key=value", securityOpt)
	}
	key, value := fields[0], fields[1]
	switch key {
	// TODO: handle other security options.
	case "apparmor":
		meta.AppArmorProfile = value
	case "seccomp":
		meta.SeccompProfile = value
	default:
		return fmt.Errorf("invalid type %s in --security-opt %s: unknown type from apparmor and seccomp", key, securityOpt)
	}
	return nil
}

// fieldsASCII is similar to strings.Fields but only allows ASCII whitespaces
func fieldsASCII(s string) []string {
	fn := func(r rune) bool {
		switch r {
		case '\t', '\n', '\f', '\r', ' ':
			return true
		}
		return false
	}
	return strings.FieldsFunc(s, fn)
}

func parsePSOutput(output []byte, pids []int) (*types.ContainerProcessList, error) {
	procList := &types.ContainerProcessList{}

	lines := strings.Split(string(output), "\n")
	procList.Titles = fieldsASCII(lines[0])

	pidIndex := -1
	for i, name := range procList.Titles {
		if name == "PID" {
			pidIndex = i
		}
	}
	if pidIndex == -1 {
		return nil, fmt.Errorf("Couldn't find PID field in ps output")
	}

	// loop through the output and extract the PID from each line
	for _, line := range lines[1:] {
		if len(line) == 0 {
			continue
		}
		fields := fieldsASCII(line)
		p, err := strconv.Atoi(fields[pidIndex])
		if err != nil {
			return nil, fmt.Errorf("Unexpected pid '%s': %s", fields[pidIndex], err)
		}

		for _, pid := range pids {
			if pid == p {
				// Make sure number of fields equals number of header titles
				// merging "overhanging" fields
				process := fields[:len(procList.Titles)-1]
				process = append(process, strings.Join(fields[len(procList.Titles)-1:], " "))
				procList.Processes = append(procList.Processes, process)
			}
		}
	}
	return procList, nil
}
