// Copyright 2016 The go-ddmchain Authors
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

package ddmclient

import "github.com/ddmchain/go-ddmchain"

// Verify that Client implements the ddmchain interfaces.
var (
	_ = ddmchain.ChainReader(&Client{})
	_ = ddmchain.TransactionReader(&Client{})
	_ = ddmchain.ChainStateReader(&Client{})
	_ = ddmchain.ChainSyncReader(&Client{})
	_ = ddmchain.ContractCaller(&Client{})
	_ = ddmchain.GasEstimator(&Client{})
	_ = ddmchain.GasPricer(&Client{})
	_ = ddmchain.LogFilterer(&Client{})
	_ = ddmchain.PendingStateReader(&Client{})
	// _ = ddmchain.PendingStateEventer(&Client{})
	_ = ddmchain.PendingContractCaller(&Client{})
)
