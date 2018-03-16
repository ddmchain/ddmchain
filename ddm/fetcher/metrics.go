// Copyright 2015 The go-ddmchain Authors
// This file is part of the go-ddmchain library.
//
// The go-ddmchain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ddmchain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ddmchain library. If not, see <http://www.gnu.org/licenses/>.

// Contains the metrics collected by the fetcher.

package fetcher

import (
	"github.com/ddmchain/go-ddmchain/metrics"
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
