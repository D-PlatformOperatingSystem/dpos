package net

import (
	"strings"

	"github.com/libp2p/go-libp2p/config"
	ma "github.com/multiformats/go-multiaddr"
)

// WithRelayAddrs     relay         addrs  ，      ，           relay          ，
//             relay    ，    relay        。
/*
				        R   ：/ip4/116.63.171.186/tcp/13801/p2p/16Uiu2HAm1Qgogf1vHj7WV7d8YmVNPw6PzRuYEnJbQsMDmDWtLoEM
				NAT          A ：/ip4/192.168.1.101/tcp/13801/p2p/16Uiu2HAkw6w2YVenbCLAXHv2XVo1kh945GVxQrpm5Y6z2kE3eFSg
			       ：A--->R  A       R
				   ：A           ：
			[/ip4/116.63.171.186/tcp/13801/p2p/16Uiu2HAm1Qgogf1vHj7WV7d8YmVNPw6PzRuYEnJbQsMDmDWtLoEM/p2p-circuit/ip4/192.168.1.101/tcp
		    /13801/p2p/16Uiu2HAkw6w2YVenbCLAXHv2XVo1kh945GVxQrpm5Y6z2kE3eFSg]
		           ：A          p2p-circuit
		           ：         NAT     NAT     ，       PID 16Uiu2HAkw6w2YVenbCLAXHv2XVo1kh945GVxQrpm5Y6z2kE3eFSg A  ，
	                               p2p-circuit          A

*/
func WithRelayAddrs(relays []string) config.AddrsFactory { //    relay
	return func(addrs []ma.Multiaddr) []ma.Multiaddr {
		if len(relays) == 0 {
			return addrs
		}

		var relayAddrs []ma.Multiaddr
		for _, a := range addrs {
			if strings.Contains(a.String(), "/p2p-circuit") {
				continue
			}
			for _, relay := range relays {
				relayAddr, err := ma.NewMultiaddr(relay + "/p2p-circuit" + a.String())
				if err != nil {
					log.Error("Failed to create multiaddress for relay node: %v", err)
				} else {
					relayAddrs = append(relayAddrs, relayAddr)
				}
			}

		}

		if len(relayAddrs) == 0 {
			log.Warn("no relay addresses")
			return addrs
		}
		return append(addrs, relayAddrs...)
	}
}
