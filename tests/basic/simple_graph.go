// Copyright 2023 NJWS Inc.

package basic

import (
	"github.com/foliagecp/easyjson"
	lg "github.com/foliagecp/sdk/statefun/logger"

	"github.com/foliagecp/sdk/statefun"
	"github.com/foliagecp/sdk/statefun/plugins"
	"github.com/foliagecp/sdk/statefun/system"
)

func CreateTestGraph(runtime *statefun.Runtime) {
	lg.Logln(lg.DebugLevel, ">>> Test started: simple graph creation")

	system.MsgOnErrorReturn(runtime.Request(plugins.GolangLocalRequest, "functions.graph.ll.api.vertex.create", "root", easyjson.NewJSONObject().GetPtr(), nil))
	system.MsgOnErrorReturn(runtime.Request(plugins.GolangLocalRequest, "functions.graph.ll.api.vertex.create", "a", easyjson.NewJSONObject().GetPtr(), nil))
	system.MsgOnErrorReturn(runtime.Request(plugins.GolangLocalRequest, "functions.graph.ll.api.vertex.create", "b", easyjson.NewJSONObject().GetPtr(), nil))
	system.MsgOnErrorReturn(runtime.Request(plugins.GolangLocalRequest, "functions.graph.ll.api.vertex.create", "c", easyjson.NewJSONObject().GetPtr(), nil))
	system.MsgOnErrorReturn(runtime.Request(plugins.GolangLocalRequest, "functions.graph.ll.api.vertex.create", "d", easyjson.NewJSONObject().GetPtr(), nil))
	system.MsgOnErrorReturn(runtime.Request(plugins.GolangLocalRequest, "functions.graph.ll.api.vertex.create", "e", easyjson.NewJSONObject().GetPtr(), nil))
	system.MsgOnErrorReturn(runtime.Request(plugins.GolangLocalRequest, "functions.graph.ll.api.vertex.create", "f", easyjson.NewJSONObject().GetPtr(), nil))
	system.MsgOnErrorReturn(runtime.Request(plugins.GolangLocalRequest, "functions.graph.ll.api.vertex.create", "g", easyjson.NewJSONObject().GetPtr(), nil))
	system.MsgOnErrorReturn(runtime.Request(plugins.GolangLocalRequest, "functions.graph.ll.api.vertex.create", "h", easyjson.NewJSONObject().GetPtr(), nil))

	var v easyjson.JSON

	v = easyjson.NewJSONObject()
	v.SetByPath("descendant_uuid", easyjson.NewJSON("a"))
	v.SetByPath("link_type", easyjson.NewJSON("type1"))
	v.SetByPath("link_body.tags", easyjson.JSONFromArray([]string{"t1", "t2"}))
	system.MsgOnErrorReturn(runtime.Request(plugins.GolangLocalRequest, "functions.graph.ll.api.link.create", "root", &v, nil))

	v = easyjson.NewJSONObject()
	v.SetByPath("descendant_uuid", easyjson.NewJSON("a"))
	v.SetByPath("link_type", easyjson.NewJSON("type2"))
	v.SetByPath("link_body.tags", easyjson.JSONFromArray([]string{"t2", "t4"}))
	system.MsgOnErrorReturn(runtime.Request(plugins.GolangLocalRequest, "functions.graph.ll.api.link.create", "root", &v, nil))

	v = easyjson.NewJSONObject()
	v.SetByPath("descendant_uuid", easyjson.NewJSON("b"))
	v.SetByPath("link_type", easyjson.NewJSON("type2"))
	v.SetByPath("link_body.tags", easyjson.JSONFromArray([]string{"t2"}))
	system.MsgOnErrorReturn(runtime.Request(plugins.GolangLocalRequest, "functions.graph.ll.api.link.create", "root", &v, nil))

	v = easyjson.NewJSONObject()
	v.SetByPath("descendant_uuid", easyjson.NewJSON("c"))
	v.SetByPath("link_type", easyjson.NewJSON("type1"))
	v.SetByPath("link_body.tags", easyjson.NewJSONObject())
	system.MsgOnErrorReturn(runtime.Request(plugins.GolangLocalRequest, "functions.graph.ll.api.link.create", "root", &v, nil))

	v = easyjson.NewJSONObject()
	v.SetByPath("descendant_uuid", easyjson.NewJSON("e"))
	v.SetByPath("link_type", easyjson.NewJSON("type3"))
	v.SetByPath("link_body.tags", easyjson.JSONFromArray([]string{"t3"}))
	system.MsgOnErrorReturn(runtime.Request(plugins.GolangLocalRequest, "functions.graph.ll.api.link.create", "a", &v, nil))

	v = easyjson.NewJSONObject()
	v.SetByPath("descendant_uuid", easyjson.NewJSON("e"))
	v.SetByPath("link_type", easyjson.NewJSON("type4"))
	v.SetByPath("link_body.tags", easyjson.JSONFromArray([]string{"t1", "t2", "t3"}))
	system.MsgOnErrorReturn(runtime.Request(plugins.GolangLocalRequest, "functions.graph.ll.api.link.create", "b", &v, nil))

	v = easyjson.NewJSONObject()
	v.SetByPath("descendant_uuid", easyjson.NewJSON("d"))
	v.SetByPath("link_type", easyjson.NewJSON("type3"))
	v.SetByPath("link_body.tags", easyjson.JSONFromArray([]string{"t1"}))
	system.MsgOnErrorReturn(runtime.Request(plugins.GolangLocalRequest, "functions.graph.ll.api.link.create", "c", &v, nil))

	v = easyjson.NewJSONObject()
	v.SetByPath("descendant_uuid", easyjson.NewJSON("b"))
	v.SetByPath("link_type", easyjson.NewJSON("type1"))
	v.SetByPath("link_body.tags", easyjson.JSONFromArray([]string{"t1", "t3"}))
	system.MsgOnErrorReturn(runtime.Request(plugins.GolangLocalRequest, "functions.graph.ll.api.link.create", "d", &v, nil))

	v = easyjson.NewJSONObject()
	v.SetByPath("descendant_uuid", easyjson.NewJSON("b"))
	v.SetByPath("link_type", easyjson.NewJSON("type2"))
	v.SetByPath("link_body.tags", easyjson.JSONFromArray([]string{"t4"}))
	system.MsgOnErrorReturn(runtime.Request(plugins.GolangLocalRequest, "functions.graph.ll.api.link.create", "e", &v, nil))

	v = easyjson.NewJSONObject()
	v.SetByPath("descendant_uuid", easyjson.NewJSON("f"))
	v.SetByPath("link_type", easyjson.NewJSON("type1"))
	v.SetByPath("link_body.tags", easyjson.JSONFromArray([]string{"t1", "t4"}))
	system.MsgOnErrorReturn(runtime.Request(plugins.GolangLocalRequest, "functions.graph.ll.api.link.create", "e", &v, nil))

	v = easyjson.NewJSONObject()
	v.SetByPath("descendant_uuid", easyjson.NewJSON("g"))
	v.SetByPath("link_type", easyjson.NewJSON("type5"))
	v.SetByPath("link_body.tags", easyjson.JSONFromArray([]string{"t1", "t2", "t3", "t4"}))
	system.MsgOnErrorReturn(runtime.Request(plugins.GolangLocalRequest, "functions.graph.ll.api.link.create", "f", &v, nil))

	v = easyjson.NewJSONObject()
	v.SetByPath("descendant_uuid", easyjson.NewJSON("d"))
	v.SetByPath("link_type", easyjson.NewJSON("type2"))
	v.SetByPath("link_body.tags", easyjson.JSONFromArray([]string{"t5"}))
	system.MsgOnErrorReturn(runtime.Request(plugins.GolangLocalRequest, "functions.graph.ll.api.link.create", "g", &v, nil))

	v = easyjson.NewJSONObject()
	v.SetByPath("descendant_uuid", easyjson.NewJSON("h"))
	v.SetByPath("link_type", easyjson.NewJSON("type2"))
	v.SetByPath("link_body.tags", easyjson.JSONFromArray([]string{}))
	system.MsgOnErrorReturn(runtime.Request(plugins.GolangLocalRequest, "functions.graph.ll.api.link.create", "g", &v, nil))

	lg.Logln(lg.DebugLevel, "<<< Test ended: simple graph creation")
}
