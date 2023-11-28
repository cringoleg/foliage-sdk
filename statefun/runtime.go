// Copyright 2023 NJWS Inc.

// Foliage primary statefun package.
// Provides all everything that is needed for Foliage stateful functions and Foliage applications
package statefun

import (
	"context"
	"fmt"
	"slices"
	"sync/atomic"
	"time"

	lg "github.com/foliagecp/sdk/statefun/logger"

	"github.com/foliagecp/sdk/statefun/cache"
	"github.com/foliagecp/sdk/statefun/system"
	"github.com/nats-io/nats.go"
)

type Runtime struct {
	config     RuntimeConfig
	nc         *nats.Conn
	js         nats.JetStreamContext
	kv         nats.KeyValue
	cacheStore *cache.Store

	registeredFunctionTypes map[string]*FunctionType

	gt0  int64 // Global time 0 - time of the very first message receving by any function type
	glce int64 // Global last call ended - time of last call of last function handling id of any function type
	gc   int64 // Global counter - max total id handlers for all function types
}

func NewRuntime(config RuntimeConfig) (r *Runtime, err error) {
	r = &Runtime{
		config:                  config,
		registeredFunctionTypes: make(map[string]*FunctionType),
	}

	r.nc, err = nats.Connect(config.natsURL)
	if err != nil {
		return
	}

	r.js, err = r.nc.JetStream(nats.PublishAsyncMaxPending(256))
	if err != nil {
		return
	}

	// Create application key value store bucket if does not exist --
	kvExists := false
	for name := range r.js.KeyValueStoreNames() {
		if name == "KV_"+config.keyValueStoreBucketName {
			r.kv, err = r.js.KeyValue(config.keyValueStoreBucketName)
			if err != nil {
				return
			}
			kvExists = true
		}
	}
	if !kvExists {
		r.kv, err = r.js.CreateKeyValue(&nats.KeyValueConfig{
			Bucket: config.keyValueStoreBucketName,
		})
		if err != nil {
			return
		}
		kvExists = true
	}
	if !kvExists {
		err = fmt.Errorf("Nats KV was not inited")
		return
	}
	// --------------------------------------------------------------

	return
}

func (r *Runtime) Start(cacheConfig *cache.Config, onAfterStart func(runtime *Runtime) error) (err error) {
	// Create streams if does not exist ------------------------------
	/* Each stream contains a single subject (topic).
	 * Differently named stream with overlapping subjects cannot exist!
	 */
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var existingStreams []string
	for info := range r.js.StreamsInfo(nats.Context(ctx)) {
		existingStreams = append(existingStreams, info.Config.Name)
	}
	for _, functionType := range r.registeredFunctionTypes {
		if !slices.Contains(existingStreams, functionType.getStreamName()) {
			_, err := r.js.AddStream(&nats.StreamConfig{
				Name:     functionType.getStreamName(),
				Subjects: []string{functionType.subject},
			})
			system.MsgOnErrorReturn(err)
		}
	}
	// --------------------------------------------------------------

	lg.Logln(lg.TraceLevel, "Initializing the cache store...")
	r.cacheStore = cache.NewCacheStore(context.Background(), cacheConfig, r.kv)
	lg.Logln(lg.TraceLevel, "Cache store inited!")

	singleInstanceFunctionsRevisions := map[string]uint64{}
	singleInstanceFunctionsLockUpdater := func(singleInstanceFunctionsRevisions map[string]uint64) {
		if len(singleInstanceFunctionsRevisions) > 0 {
			for {
				time.Sleep(time.Duration(r.config.kvMutexLifeTimeSec) / 2 * time.Second)
				for k, revId := range singleInstanceFunctionsRevisions {
					newRevId, err := KeyMutexLockUpdate(r, k, revId)
					if err != nil {
						lg.Logf(lg.ErrorLevel, "KeyMutexLockUpdate for single instance function with stream=%s failed", k)
					} else {
						singleInstanceFunctionsRevisions[k] = newRevId
					}
				}
			}
		}
	}

	// Start function subscriptions ---------------------------------
	for _, ft := range r.registeredFunctionTypes {
		if !ft.config.multipleInstancesAllowed {
			revId, err := KeyMutexLock(r, ft.getStreamName(), true)
			if err != nil {
				if err == mutexLockedError {
					lg.Logf(lg.WarnLevel, "Function type %s is already running somewhere and multipleInstancesAllowed==false, skipping", ft.name)
					continue
				} else {
					return err
				}
			}
			singleInstanceFunctionsRevisions[ft.getStreamName()] = revId
		}

		system.MsgOnErrorReturn(AddSignalSourceJetstreamQueuePushConsumer(ft))
		if ft.config.serviceActive {
			system.MsgOnErrorReturn(AddRequestSourceNatsCore(ft))
		}
	}
	// --------------------------------------------------------------

	go singleInstanceFunctionsLockUpdater(singleInstanceFunctionsRevisions)

	if onAfterStart != nil {
		system.MsgOnErrorReturn(onAfterStart(r))
	}
	system.MsgOnErrorReturn(r.runGarbageCellector())

	return
}

func (r *Runtime) runGarbageCellector() (err error) {
	for {
		// Start function subscriptions ---------------------------------
		var totalIdsGrbageCollected int
		var totalIDHandlersRunning int
		for _, ft := range r.registeredFunctionTypes {
			n1, n2 := ft.gc(r.config.functionTypeIDLifetimeMs)
			totalIdsGrbageCollected += n1
			totalIDHandlersRunning += n2
		}
		if totalIdsGrbageCollected > 0 && totalIDHandlersRunning == 0 {
			// Result time output -----------------------------------------------------------------
			if totalIDHandlersRunning == 0 {
				glce := atomic.LoadInt64(&r.glce)
				gt0 := atomic.LoadInt64(&r.gt0)
				gc := atomic.LoadInt64(&r.gc)

				dt := glce - gt0

				if gc > 0 && dt > 0 {
					lg.Logf(lg.TraceLevel, "!!!!!!!!!!!!!!!!! %d runs, total time (ns/ms): %d/%d, function dt (ns/ms): %d/%d -> %dHz\n", gc, dt, dt/1000000, dt/gc, dt/gc/1000000, gc*1000000000/dt)
					atomic.StoreInt64(&r.gc, 0)
				}
				// ------------------------------------------------------------------------------------
			}
			// --------------------------------------------------------------
		}
		time.Sleep(1 * time.Second)
	}
}
