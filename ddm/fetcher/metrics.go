
package fetcher

import (
	"github.com/ddmchain/go-ddmchain/rhythm"
)

var (
	propAnnounceInMeter   = metrics.NewMeter("ddm/fetcher/prop/announces/in")
	propAnnounceOutTimer  = metrics.NewTimer("ddm/fetcher/prop/announces/out")
	propAnnounceDropMeter = metrics.NewMeter("ddm/fetcher/prop/announces/drop")
	propAnnounceDOSMeter  = metrics.NewMeter("ddm/fetcher/prop/announces/dos")

	propBroadcastInMeter   = metrics.NewMeter("ddm/fetcher/prop/broadcasts/in")
	propBroadcastOutTimer  = metrics.NewTimer("ddm/fetcher/prop/broadcasts/out")
	propBroadcastDropMeter = metrics.NewMeter("ddm/fetcher/prop/broadcasts/drop")
	propBroadcastDOSMeter  = metrics.NewMeter("ddm/fetcher/prop/broadcasts/dos")

	headerFetchMeter = metrics.NewMeter("ddm/fetcher/fetch/headers")
	bodyFetchMeter   = metrics.NewMeter("ddm/fetcher/fetch/bodies")

	headerFilterInMeter  = metrics.NewMeter("ddm/fetcher/filter/headers/in")
	headerFilterOutMeter = metrics.NewMeter("ddm/fetcher/filter/headers/out")
	bodyFilterInMeter    = metrics.NewMeter("ddm/fetcher/filter/bodies/in")
	bodyFilterOutMeter   = metrics.NewMeter("ddm/fetcher/filter/bodies/out")
)
