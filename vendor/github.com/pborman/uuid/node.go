
package uuid

import (
	"net"
	"sync"
)

var (
	nodeMu     sync.Mutex
	interfaces []net.Interface 
	ifname     string          
	nodeID     []byte          
)

func NodeInterface() string {
	defer nodeMu.Unlock()
	nodeMu.Lock()
	return ifname
}

func SetNodeInterface(name string) bool {
	defer nodeMu.Unlock()
	nodeMu.Lock()
	return setNodeInterface(name)
}

func setNodeInterface(name string) bool {
	if interfaces == nil {
		var err error
		interfaces, err = net.Interfaces()
		if err != nil && name != "" {
			return false
		}
	}

	for _, ifs := range interfaces {
		if len(ifs.HardwareAddr) >= 6 && (name == "" || name == ifs.Name) {
			if setNodeID(ifs.HardwareAddr) {
				ifname = ifs.Name
				return true
			}
		}
	}

	if name == "" {
		if nodeID == nil {
			nodeID = make([]byte, 6)
		}
		randomBits(nodeID)
		return true
	}
	return false
}

func NodeID() []byte {
	defer nodeMu.Unlock()
	nodeMu.Lock()
	if nodeID == nil {
		setNodeInterface("")
	}
	nid := make([]byte, 6)
	copy(nid, nodeID)
	return nid
}

func SetNodeID(id []byte) bool {
	defer nodeMu.Unlock()
	nodeMu.Lock()
	if setNodeID(id) {
		ifname = "user"
		return true
	}
	return false
}

func setNodeID(id []byte) bool {
	if len(id) < 6 {
		return false
	}
	if nodeID == nil {
		nodeID = make([]byte, 6)
	}
	copy(nodeID, id)
	return true
}

func (uuid UUID) NodeID() []byte {
	if len(uuid) != 16 {
		return nil
	}
	node := make([]byte, 6)
	copy(node, uuid[10:])
	return node
}
