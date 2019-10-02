package proxycfg

import (
	"context"

	"github.com/hashicorp/consul/agent/structs"
	"github.com/mitchellh/copystructure"
)

type configSnapshotConnectProxy struct {
	Leaf                     *structs.IssuedCert
	DiscoveryChain           map[string]*structs.CompiledDiscoveryChain // this is keyed by the Upstream.Identifier(), not the chain name
	WatchedUpstreams         map[string]map[string]context.CancelFunc
	WatchedUpstreamEndpoints map[string]map[string]structs.CheckServiceNodes
	WatchedGateways          map[string]map[string]context.CancelFunc
	WatchedGatewayEndpoints  map[string]map[string]structs.CheckServiceNodes
	WatchedServiceChecks     map[string][]structs.CheckType // TODO: missing garbage collection

	UpstreamEndpoints map[string]structs.CheckServiceNodes // DEPRECATED:see:WatchedUpstreamEndpoints
}

func (c *configSnapshotConnectProxy) IsEmpty() bool {
	if c == nil {
		return true
	}
	return c.Leaf == nil &&
		len(c.DiscoveryChain) == 0 &&
		len(c.WatchedUpstreams) == 0 &&
		len(c.WatchedUpstreamEndpoints) == 0 &&
		len(c.WatchedGateways) == 0 &&
		len(c.WatchedGatewayEndpoints) == 0 &&
		len(c.WatchedServiceChecks) == 0 &&
		len(c.UpstreamEndpoints) == 0
}

type configSnapshotMeshGateway struct {
	WatchedServices    map[string]context.CancelFunc
	WatchedServicesSet bool
	WatchedDatacenters map[string]context.CancelFunc
	ServiceGroups      map[string]structs.CheckServiceNodes
	ServiceResolvers   map[string]*structs.ServiceResolverConfigEntry
	GatewayGroups      map[string]structs.CheckServiceNodes
}

func (c *configSnapshotMeshGateway) IsEmpty() bool {
	if c == nil {
		return true
	}
	return len(c.WatchedServices) == 0 &&
		!c.WatchedServicesSet &&
		len(c.WatchedDatacenters) == 0 &&
		len(c.ServiceGroups) == 0 &&
		len(c.ServiceResolvers) == 0 &&
		len(c.GatewayGroups) == 0
}

// ConfigSnapshot captures all the resulting config needed for a proxy instance.
// It is meant to be point-in-time coherent and is used to deliver the current
// config state to observers who need it to be pushed in (e.g. XDS server).
type ConfigSnapshot struct {
	Kind            structs.ServiceKind
	Service         string
	ProxyID         string
	Address         string
	Port            int
	TaggedAddresses map[string]structs.ServiceAddress
	Proxy           structs.ConnectProxyConfig
	Datacenter      string
	Roots           *structs.IndexedCARoots

	// connect-proxy specific
	ConnectProxy configSnapshotConnectProxy

	// mesh-gateway specific
	MeshGateway configSnapshotMeshGateway

	// Skip intentions for now as we don't push those down yet, just pre-warm them.
}

// Valid returns whether or not the snapshot has all required fields filled yet.
func (s *ConfigSnapshot) Valid() bool {
	switch s.Kind {
	case structs.ServiceKindConnectProxy:
		return s.Roots != nil && s.ConnectProxy.Leaf != nil
	case structs.ServiceKindMeshGateway:
		return s.Roots != nil && s.MeshGateway.WatchedServicesSet
	default:
		return false
	}
}

// Clone makes a deep copy of the snapshot we can send to other goroutines
// without worrying that they will racily read or mutate shared maps etc.
func (s *ConfigSnapshot) Clone() (*ConfigSnapshot, error) {
	snapCopy, err := copystructure.Copy(s)
	if err != nil {
		return nil, err
	}

	snap := snapCopy.(*ConfigSnapshot)

	// nil these out as anything receiving one of these clones does not need them and should never "cancel" our watches
	switch s.Kind {
	case structs.ServiceKindConnectProxy:
		snap.ConnectProxy.WatchedUpstreams = nil
		snap.ConnectProxy.WatchedGateways = nil
	case structs.ServiceKindMeshGateway:
		snap.MeshGateway.WatchedDatacenters = nil
		snap.MeshGateway.WatchedServices = nil
	}

	return snap, nil
}
