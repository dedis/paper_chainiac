#!/usr/bin/env bash

for path in :github.com/dedis/cothority/sda:github.com/dedis/onet: \
    :github.com/dedis/cothority/network:github.com/dedis/onet/network: \
    :github.com/dedis/cothority/log:github.com/dedis/onet/log: \
    :github.com/dedis/cothority/monitor:github.com/dedis/onet/simul/monitor: \
    :github.com/dedis/cothority/crypto:github.com/dedis/onet/crypto: \
    :github.com/dedis/cothority/protocols/manage:github.com/dedis/cothority/manage:; do
        find . -name "*go" | xargs perl -pi -e "s$path"
done

find . -name "*go" | xargs perl -pi -e s:github.com/dedis/onet:gopkg.in/dedis/onet.v1:

for oldnew in sda\\.:onet. network\\.Body:network.Message \
	onet\\.ProtocolRegisterName:onet.GlobalProtocolRegister \
	network\\.RegisterHandler:network.RegisterMessage \
	ServerIdentity\\.Addresses:ServerIdentity.Address \
	CreateProtocolService:CreateProtocol \
	CreateProtocolSDA:CreateProtocol \
    RegisterPacketType:RegisterMessage \
    network\\.Packet:network.Envelope sda\\.Conode:onet.Server \
    UnmarshalRegistered:Unmarshal MarshalRegisteredType:Marshal ; do
    	echo replacing $oldnew
        find . -name "*go" | xargs -n 1 perl -pi -e s:$oldnew:g
done

