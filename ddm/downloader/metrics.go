
package downloader

import (
	"github.com/ddmchain/go-ddmchain/rhythm"
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
