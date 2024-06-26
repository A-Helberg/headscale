package util

import (
	"net/netip"

	"github.com/google/go-cmp/cmp"
	"tailscale.com/types/key"
)

var PrefixComparer = cmp.Comparer(func(x, y netip.Prefix) bool {
	return x == y
})

var IPComparer = cmp.Comparer(func(x, y netip.Addr) bool {
	return x.Compare(y) == 0
})

var MkeyComparer = cmp.Comparer(func(x, y key.MachinePublic) bool {
	return x.String() == y.String()
})

var NkeyComparer = cmp.Comparer(func(x, y key.NodePublic) bool {
	return x.String() == y.String()
})

var DkeyComparer = cmp.Comparer(func(x, y key.DiscoPublic) bool {
	return x.String() == y.String()
})

var Comparers []cmp.Option = []cmp.Option{
	IPComparer, PrefixComparer, MkeyComparer, NkeyComparer, DkeyComparer,
}
