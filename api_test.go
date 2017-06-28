package debianupdate

import (
	"testing"

	"github.com/dedis/paper_chainiac/skipchain"
	"github.com/stretchr/testify/require"
	"gopkg.in/dedis/onet.v1"
	"gopkg.in/dedis/onet.v1/log"
)

func TestClient_LatestUpdates(t *testing.T) {
	local := onet.NewLocalTest()
	defer local.CloseAll()
	_, roster, s := local.MakeHELS(5, debianUpdateService)
	service := s.(*DebianUpdate)

	cpr, err := service.CreateRepository(nil,
		&CreateRepository{roster, chain1.blocks[0].release, 2, 10})
	log.ErrFatal(err)
	sc := cpr.(*CreateRepositoryRet).RepositoryChain

	upr, err := service.UpdateRepository(nil,
		&UpdateRepository{sc, chain1.blocks[1].release})
	log.ErrFatal(err)
	sc2 := upr.(*UpdateRepositoryRet).RepositoryChain

	client := NewClient(roster)
	lbret, err := client.LatestUpdates([]skipchain.SkipBlockID{sc.Data.Hash})
	log.ErrFatal(err)
	require.Equal(t, 1, len(lbret.Updates))
	require.Equal(t, sc2.Data.Hash, lbret.Updates[0][1].Hash)

	cpr, err = service.CreateRepository(nil,
		&CreateRepository{roster, chain2.blocks[0].release, 2, 10})
	log.ErrFatal(err)
	sc3 := cpr.(*CreateRepositoryRet).RepositoryChain

	upr, err = service.UpdateRepository(nil,
		&UpdateRepository{sc3, chain2.blocks[1].release})
	log.ErrFatal(err)
	sc4 := upr.(*UpdateRepositoryRet).RepositoryChain

	lbret, err = client.LatestUpdates([]skipchain.SkipBlockID{sc.Data.Hash,
		sc3.Data.Hash})
	log.ErrFatal(err)
	require.Equal(t, 2, len(lbret.Updates))
	require.Equal(t, 2, len(lbret.Updates[0]))
	require.Equal(t, 2, len(lbret.Updates[1]))
	require.Equal(t, sc2.Data.Hash, lbret.Updates[0][1].Hash)
	require.Equal(t, sc4.Data.Hash, lbret.Updates[1][1].Hash)

	lbret, err = client.LatestUpdates([]skipchain.SkipBlockID{sc2.Data.Hash,
		sc3.Data.Hash})
	log.ErrFatal(err)
	require.Equal(t, 1, len(lbret.Updates))
	require.Equal(t, 2, len(lbret.Updates[0]))
}
