package types

import (
	"net/netip"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"tailscale.com/tailcfg"
	"tailscale.com/types/key"
)

func Test_NodeCanAccess(t *testing.T) {
	tests := []struct {
		name  string
		node1 Node
		node2 Node
		rules []tailcfg.FilterRule
		want  bool
	}{
		{
			name: "no-rules",
			node1: Node{
				IPAddresses: []netip.Addr{netip.MustParseAddr("10.0.0.1")},
			},
			node2: Node{
				IPAddresses: []netip.Addr{netip.MustParseAddr("10.0.0.2")},
			},
			rules: []tailcfg.FilterRule{},
			want:  false,
		},
		{
			name: "wildcard",
			node1: Node{
				IPAddresses: []netip.Addr{netip.MustParseAddr("10.0.0.1")},
			},
			node2: Node{
				IPAddresses: []netip.Addr{netip.MustParseAddr("10.0.0.2")},
			},
			rules: []tailcfg.FilterRule{
				{
					SrcIPs: []string{"*"},
					DstPorts: []tailcfg.NetPortRange{
						{
							IP:    "*",
							Ports: tailcfg.PortRangeAny,
						},
					},
				},
			},
			want: true,
		},
		{
			name: "other-cant-access-src",
			node1: Node{
				IPAddresses: []netip.Addr{netip.MustParseAddr("100.64.0.1")},
			},
			node2: Node{
				IPAddresses: []netip.Addr{netip.MustParseAddr("100.64.0.3")},
			},
			rules: []tailcfg.FilterRule{
				{
					SrcIPs: []string{"100.64.0.2/32"},
					DstPorts: []tailcfg.NetPortRange{
						{IP: "100.64.0.3/32", Ports: tailcfg.PortRangeAny},
					},
				},
			},
			want: false,
		},
		{
			name: "dest-cant-access-src",
			node1: Node{
				IPAddresses: []netip.Addr{netip.MustParseAddr("100.64.0.3")},
			},
			node2: Node{
				IPAddresses: []netip.Addr{netip.MustParseAddr("100.64.0.2")},
			},
			rules: []tailcfg.FilterRule{
				{
					SrcIPs: []string{"100.64.0.2/32"},
					DstPorts: []tailcfg.NetPortRange{
						{IP: "100.64.0.3/32", Ports: tailcfg.PortRangeAny},
					},
				},
			},
			want: false,
		},
		{
			name: "src-can-access-dest",
			node1: Node{
				IPAddresses: []netip.Addr{netip.MustParseAddr("100.64.0.2")},
			},
			node2: Node{
				IPAddresses: []netip.Addr{netip.MustParseAddr("100.64.0.3")},
			},
			rules: []tailcfg.FilterRule{
				{
					SrcIPs: []string{"100.64.0.2/32"},
					DstPorts: []tailcfg.NetPortRange{
						{IP: "100.64.0.3/32", Ports: tailcfg.PortRangeAny},
					},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.node1.CanAccess(tt.rules, &tt.node2)

			if got != tt.want {
				t.Errorf("canAccess() failed: want (%t), got (%t)", tt.want, got)
			}
		})
	}
}

func TestNodeAddressesOrder(t *testing.T) {
	machineAddresses := NodeAddresses{
		netip.MustParseAddr("2001:db8::2"),
		netip.MustParseAddr("100.64.0.2"),
		netip.MustParseAddr("2001:db8::1"),
		netip.MustParseAddr("100.64.0.1"),
	}

	strSlice := machineAddresses.StringSlice()
	expected := []string{
		"100.64.0.1",
		"100.64.0.2",
		"2001:db8::1",
		"2001:db8::2",
	}

	if len(strSlice) != len(expected) {
		t.Fatalf("unexpected slice length: got %v, want %v", len(strSlice), len(expected))
	}
	for i, addr := range strSlice {
		if addr != expected[i] {
			t.Errorf("unexpected address at index %v: got %v, want %v", i, addr, expected[i])
		}
	}
}

func TestNodeFQDN(t *testing.T) {
	tests := []struct {
		name    string
		node    Node
		dns     tailcfg.DNSConfig
		domain  string
		want    string
		wantErr string
	}{
		{
			name: "all-set",
			node: Node{
				GivenName: "test",
				User: User{
					Name: "user",
				},
			},
			dns: tailcfg.DNSConfig{
				Proxied: true,
			},
			domain: "example.com",
			want:   "test.user.example.com",
		},
		{
			name: "no-given-name",
			node: Node{
				User: User{
					Name: "user",
				},
			},
			dns: tailcfg.DNSConfig{
				Proxied: true,
			},
			domain:  "example.com",
			wantErr: "failed to create valid FQDN: node has no given name",
		},
		{
			name: "no-user-name",
			node: Node{
				GivenName: "test",
				User:      User{},
			},
			dns: tailcfg.DNSConfig{
				Proxied: true,
			},
			domain:  "example.com",
			wantErr: "failed to create valid FQDN: node user has no name",
		},
		{
			name: "no-magic-dns",
			node: Node{
				GivenName: "test",
				User: User{
					Name: "user",
				},
			},
			dns: tailcfg.DNSConfig{
				Proxied: false,
			},
			domain: "example.com",
			want:   "test",
		},
		{
			name: "no-dnsconfig",
			node: Node{
				GivenName: "test",
				User: User{
					Name: "user",
				},
			},
			domain: "example.com",
			want:   "test",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.node.GetFQDN(&tc.dns, tc.domain)

			if (err != nil) && (err.Error() != tc.wantErr) {
				t.Errorf("GetFQDN() error = %s, wantErr %s", err, tc.wantErr)

				return
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GetFQDN unexpected result (-want +got):\n%s", diff)
			}
		})
	}
}

func TestPeerChangeFromMapRequest(t *testing.T) {
	nKeys := []key.NodePublic{
		key.NewNode().Public(),
		key.NewNode().Public(),
		key.NewNode().Public(),
	}

	dKeys := []key.DiscoPublic{
		key.NewDisco().Public(),
		key.NewDisco().Public(),
		key.NewDisco().Public(),
	}

	tests := []struct {
		name   string
		node   Node
		mapReq tailcfg.MapRequest
		want   tailcfg.PeerChange
	}{
		{
			name: "preferred-derp-changed",
			node: Node{
				ID:        1,
				NodeKey:   nKeys[0],
				DiscoKey:  dKeys[0],
				Endpoints: []netip.AddrPort{},
				Hostinfo: &tailcfg.Hostinfo{
					NetInfo: &tailcfg.NetInfo{
						PreferredDERP: 998,
					},
				},
			},
			mapReq: tailcfg.MapRequest{
				NodeKey:  nKeys[0],
				DiscoKey: dKeys[0],
				Hostinfo: &tailcfg.Hostinfo{
					NetInfo: &tailcfg.NetInfo{
						PreferredDERP: 999,
					},
				},
			},
			want: tailcfg.PeerChange{
				NodeID:     1,
				DERPRegion: 999,
			},
		},
		{
			name: "preferred-derp-no-changed",
			node: Node{
				ID:        1,
				NodeKey:   nKeys[0],
				DiscoKey:  dKeys[0],
				Endpoints: []netip.AddrPort{},
				Hostinfo: &tailcfg.Hostinfo{
					NetInfo: &tailcfg.NetInfo{
						PreferredDERP: 100,
					},
				},
			},
			mapReq: tailcfg.MapRequest{
				NodeKey:  nKeys[0],
				DiscoKey: dKeys[0],
				Hostinfo: &tailcfg.Hostinfo{
					NetInfo: &tailcfg.NetInfo{
						PreferredDERP: 100,
					},
				},
			},
			want: tailcfg.PeerChange{
				NodeID:     1,
				DERPRegion: 0,
			},
		},
		{
			name: "preferred-derp-no-mapreq-netinfo",
			node: Node{
				ID:        1,
				NodeKey:   nKeys[0],
				DiscoKey:  dKeys[0],
				Endpoints: []netip.AddrPort{},
				Hostinfo: &tailcfg.Hostinfo{
					NetInfo: &tailcfg.NetInfo{
						PreferredDERP: 200,
					},
				},
			},
			mapReq: tailcfg.MapRequest{
				NodeKey:  nKeys[0],
				DiscoKey: dKeys[0],
				Hostinfo: &tailcfg.Hostinfo{},
			},
			want: tailcfg.PeerChange{
				NodeID:     1,
				DERPRegion: 0,
			},
		},
		{
			name: "preferred-derp-no-node-netinfo",
			node: Node{
				ID:        1,
				NodeKey:   nKeys[0],
				DiscoKey:  dKeys[0],
				Endpoints: []netip.AddrPort{},
				Hostinfo:  &tailcfg.Hostinfo{},
			},
			mapReq: tailcfg.MapRequest{
				NodeKey:  nKeys[0],
				DiscoKey: dKeys[0],
				Hostinfo: &tailcfg.Hostinfo{
					NetInfo: &tailcfg.NetInfo{
						PreferredDERP: 200,
					},
				},
			},
			want: tailcfg.PeerChange{
				NodeID:     1,
				DERPRegion: 200,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.node.PeerChangeFromMapRequest(tc.mapReq)

			if diff := cmp.Diff(tc.want, got, cmpopts.IgnoreFields(tailcfg.PeerChange{}, "LastSeen")); diff != "" {
				t.Errorf("Patch unexpected result (-want +got):\n%s", diff)
			}
		})
	}
}
