// Foliage basic test package.
// Provides the basic example of usage of the SDK.
package main

import (
	"time"

	//"github.com/foliagecp/easyjson"
	"github.com/foliagecp/easyjson"
	graphCRUD "github.com/foliagecp/sdk/embedded/graph/crud"

	// Comment out and no not use graphDebug for resolving the cgo conflict between go-graphviz and rogchap (when --ldflags '-extldflags "-Wl,--allow-multiple-definition"' does not help)
	//graphDebug "github.com/foliagecp/sdk/embedded/graph/debug"
	"github.com/foliagecp/sdk/embedded/graph/jpgql"

	statefun "github.com/foliagecp/sdk/statefun"
	"github.com/foliagecp/sdk/statefun/cache"
	sfPlugins "github.com/foliagecp/sdk/statefun/plugins"
	"github.com/foliagecp/sdk/statefun/system"

	lg "github.com/foliagecp/sdk/statefun/logger"
)

var (
	// NatsURL - nats server url
	NatsURL string = system.GetEnvMustProceed("NATS_URL", "nats://nats:foliage@nats:4222")
	// NatsURL - nats server url
	PrometricsServerPort string = system.GetEnvMustProceed("PROMETRICS_PORT", "9901")
	// CreateSimpleGraphTest - create a simple graph on runtime start
	CreateSimpleGraphTest bool = system.GetEnvMustProceed("CREATE_SIMPLE_GRAPH_TEST", true)
	// TriggersTest - test the Foliage cmdb crud triggers
	TriggersTest bool = system.GetEnvMustProceed("TRIGGERS_TEST", true)
)

func TestFunction(executor sfPlugins.StatefunExecutor, ctx *sfPlugins.StatefunContextProcessor) {
	hubDomain := ctx.Domain.HubDomainName()
	callerDomain := ctx.Domain.GetDomainFromObjectID(ctx.Caller.ID)
	functionDomain := ctx.Domain.GetDomainFromObjectID(ctx.Self.ID)
	if ctx.Reply == nil { // Signal came
		lg.Logf(
			lg.InfoLevel,
			">>> Signal from caller %s:%s on %s:%s\n",
			ctx.Caller.Typename,
			ctx.Caller.ID,
			ctx.Self.Typename,
			ctx.Self.ID,
		)
		if functionDomain == hubDomain { // Function on HUB
			ctx.Signal(
				sfPlugins.JetstreamGlobalSignal,
				ctx.Self.Typename,
				ctx.Domain.CreateObjectIDWithDomain("leaf", ctx.Self.ID+"A", true),
				ctx.Payload,
				ctx.Options,
			)
		} else { // Function on LEAF
			if callerDomain == hubDomain { // from HUB
				ctx.Signal(
					sfPlugins.JetstreamGlobalSignal,
					ctx.Self.Typename,
					ctx.Domain.CreateObjectIDWithDomain("leaf", ctx.Self.ID+"B", true),
					ctx.Payload,
					ctx.Options,
				)
			} else { // from LEAF
				ctx.Request(
					sfPlugins.NatsCoreGlobalRequest,
					ctx.Self.Typename,
					ctx.Domain.CreateObjectIDWithHubDomain(ctx.Self.ID+"C", true),
					ctx.Payload,
					ctx.Options,
				)
			}
		}
	} else { // Request came
		lg.Logf(
			lg.InfoLevel,
			">>>>>> Request from caller %s:%s on %s:%s\n",
			ctx.Caller.Typename,
			ctx.Caller.ID,
			ctx.Self.Typename,
			ctx.Self.ID,
		)
		if functionDomain == hubDomain { // Function on HUB
			ctx.Request(
				sfPlugins.NatsCoreGlobalRequest,
				ctx.Self.Typename,
				ctx.Domain.CreateObjectIDWithDomain("leaf", ctx.Self.ID+"D", true),
				ctx.Payload,
				ctx.Options,
			)
		}
	}
}

func RegisterFunctionTypes(runtime *statefun.Runtime) {
	statefun.NewFunctionType(runtime, "domains.test", TestFunction, *statefun.NewFunctionTypeConfig().SetAllowedRequestProviders(sfPlugins.AutoRequestSelect))

	graphCRUD.RegisterAllFunctionTypes(runtime)
	//graphDebug.RegisterAllFunctionTypes(runtime)
	jpgql.RegisterAllFunctionTypes(runtime)
}

func Start() {
	system.GlobalPrometrics = system.NewPrometrics("", ":"+PrometricsServerPort)

	afterStart := func(runtime *statefun.Runtime) error {
		if runtime.Domain.Name() == runtime.Domain.HubDomainName() {
			time.Sleep(5 * time.Second) // Wait for everything to bring up
			if TriggersTest {
				RunTriggersTest(runtime)
			}
			if CreateSimpleGraphTest {
				CreateTestGraph(runtime)
			}
			time.Sleep(1 * time.Second)
			runtime.Signal(sfPlugins.JetstreamGlobalSignal, "domains.test", "foo", easyjson.NewJSONObject().GetPtr(), nil)
		}
		return nil
	}

	if runtime, err := statefun.NewRuntime(*statefun.NewRuntimeConfigSimple(NatsURL, "distributed")); err == nil {
		RegisterFunctionTypes(runtime)
		if TriggersTest {
			registerTriggerFunctions(runtime)
		}
		runtime.RegisterOnAfterStartFunction(afterStart)
		if err := runtime.Start(cache.NewCacheConfig("main_cache")); err != nil {
			lg.Logf(lg.ErrorLevel, "Cannot start due to an error: %s\n", err)
		}
	} else {
		lg.Logf(lg.ErrorLevel, "Cannot create statefun runtime due to an error: %s\n", err)
	}
}

// --------------------------------------------------------------------------------------
