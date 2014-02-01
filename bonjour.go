package airshow

import (
	"github.com/soh335/go-dnssd"
	"net"
	"strconv"
)

func searchBonjour(protocol string, addressChannel chan string) error {
	bc := make(chan *dnssd.BrowseReply)
	ctx, err := dnssd.Browse(dnssd.DNSServiceInterfaceIndexAny, protocol, bc)

	if err != nil {
		return err
	}

	defer ctx.Release()
	go dnssd.Process(ctx)

	for {
		browseReply, ok := <-bc
		if !ok {
			break
		}

		rc := make(chan *dnssd.ResolveReply)
		rctx, err := dnssd.Resolve(
			dnssd.DNSServiceFlagsForceMulticast,
			browseReply.InterfaceIndex,
			browseReply.ServiceName,
			browseReply.RegType,
			browseReply.ReplyDomain,
			rc,
		)

		if err != nil {
			return err
		}

		defer rctx.Release()
		go dnssd.Process(rctx)

		resolveReply, _ := <-rc

		qc := make(chan *dnssd.QueryRecordReply)
		qctx, err := dnssd.QueryRecord(
			dnssd.DNSServiceFlagsForceMulticast,
			resolveReply.InterfaceIndex,
			resolveReply.FullName,
			dnssd.DNSServiceType_SRV,
			dnssd.DNSServiceClass_IN,
			qc,
		)

		if err != nil {
			return err
		}

		defer qctx.Release()
		go dnssd.Process(qctx)

		queryRecordReply, _ := <-qc
		srv := queryRecordReply.SRV()

		gc := make(chan *dnssd.GetAddrInfoReply)
		gctx, err := dnssd.GetAddrInfo(
			dnssd.DNSServiceFlagsForceMulticast,
			0,
			dnssd.DNSServiceProtocol_IPv4,
			resolveReply.HostTarget,
			gc,
		)

		if err != nil {
			return err
		}

		defer gctx.Release()
		go dnssd.Process(gctx)

		getAddrInfoReply, _ := <-gc

		addressChannel <- net.JoinHostPort(getAddrInfoReply.Ip, strconv.Itoa((int)(srv.Port)))
	}

	return nil
}
