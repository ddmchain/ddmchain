// 
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

// Contains the metrics collected by the downloader.

package downloader

import (
	"github.com/ddmchain/go-ddmchain/metrics"
)

var (
	headerInMeter      = metrics.NewMeter("ddm/downloader/headers/in")
	headerReqTimer     = metrics.NewTimer("ddm/downloader/headers/req")
	headerDropMeter    = metrics.NewMeter("ddm/downloader/headers/drop")
	headerTimeoutMeter = metrics.NewMeter("ddm/downloader/headers/timeout")

	bodyInMeter      = metrics.NewMeter("ddm/downloader/bodies/in")
	bodyReqTimer     = metrics.NewTimer("ddm/downloader/bodies/req")
	bodyDropMeter    = metrics.NewMeter("ddm/downloader/bodies/drop")
	bodyTimeoutMeter = metrics.NewMeter("ddm/downloader/bodies/timeout")

	receiptInMeter      = metrics.NewMeter("ddm/downloader/receipts/in")
	receiptReqTimer     = metrics.NewTimer("ddm/downloader/receipts/req")
	receiptDropMeter    = metrics.NewMeter("ddm/downloader/receipts/drop")
	receiptTimeoutMeter = metrics.NewMeter("ddm/downloader/receipts/timeout")

	stateInMeter   = metrics.NewMeter("ddm/downloader/states/in")
	stateDropMeter = metrics.NewMeter("ddm/downloader/states/drop")
)
