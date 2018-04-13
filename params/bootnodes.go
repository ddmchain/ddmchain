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

package params

// MainnetBootnodes are the enode URLs of the P2P bootstrap nodes running on
// the main DDMchain network.
var MainnetBootnodes = []string{
	// DDMchain Foundation Go Bootnodes
	"enode://1ba792e3516547679fdddec5633a1dfe8c47e267b2ab0ce3f9701f6dc708acee2e2a53bbe9befc285ec27de3fbecba0a04530d97309832021f391c50dc083929@47.104.187.114:50303",
	"enode://8d550a867d203c34ab80e2f45bb5c8d4765707cae7f46f183924aac78131dca2cafe31005240e42c334a54715e86ff261c5bfbefba61417dc04a2415e29d5d75@47.104.191.41:50303",
	"enode://5cf36daae41378db1db9c38cad0cb94d648ee5254c84607d58060b3220869a635f3a52b0c7bd2796cedd9433410810d846cb2bd3bf507de2710a356523533965@47.104.191.146:50303",
}

// TestnetBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Ropsten test network.
var TestnetBootnodes = []string{

}

// RinkebyBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Rinkeby test network.
var RinkebyBootnodes = []string{

}

// DiscoveryV5Bootnodes are the enode URLs of the P2P bootstrap nodes for the
// experimental RLPx v5 topic-discovery network.
var DiscoveryV5Bootnodes = []string{

}
