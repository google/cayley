package rethinkdb

import (
	"fmt"
	"math/rand"
	"runtime"
	"testing"
	"time"

	"github.com/cayleygraph/cayley/graph"
	"github.com/cayleygraph/cayley/graph/graphtest"
	"github.com/cayleygraph/cayley/internal/dock"
	docker "github.com/fsouza/go-dockerclient"
)

func makeRethinkDB(t testing.TB) (graph.QuadStore, graph.Options, func()) {
	var conf dock.Config

	conf.Image = "rethinkdb:latest"
	conf.OpenStdin = true
	conf.Tty = true

	var (
		addr   string
		closer func()
	)

	if runtime.GOOS == "linux" {
		// Normal Linux bridged network mode
		addr, closer = dock.Run(t, conf)
		addr = fmt.Sprintf("%s:%d", addr, 28015)
	} else {
		// If running test on Mac (Mac for Docker) we need to bind ports to host
		// See: https://docs.docker.com/docker-for-mac/networking/#/use-cases-and-workarounds
		conf.ExposedPorts = map[docker.Port]struct{}{
			"28015/tcp": {},
		}

		randPort := func() int {
			rand.Seed(time.Now().Unix())
			return rand.Intn(1000) + 45000
		}()

		conf.PortBindings = map[docker.Port][]docker.PortBinding{
			"28015/tcp": []docker.PortBinding{
				{
					HostPort: fmt.Sprintf("%d", randPort),
				},
			},
		}

		_, closer = dock.Run(t, conf)
		addr = fmt.Sprintf("%s:%d", "localhost", randPort)
	}

	// Retry connections, docker container might not be ready
	var err error
	for i := 0; i < 10; i++ {
		err = createNewRethinkDBGraph(addr, nil)
		if err == nil {
			break
		}
		time.Sleep(time.Second * 1)
	}

	if err != nil {
		closer()
		t.Fatal(err)
	}

	qs, err := newQuadStore(addr, nil)
	if err != nil {
		closer()
		t.Fatal(err)
	}
	return qs, nil, func() {
		qs.Close()
		closer()
	}
}

func TestRethinkDBAll(t *testing.T) {
	graphtest.TestAll(t, makeRethinkDB, &graphtest.Config{
		TimeInMs:                true,
		TimeRound:               true,
		OptimizesComparison:     true,
		SkipDeletedFromIterator: true,
		SkipIntHorizon:          true,
	})
}
