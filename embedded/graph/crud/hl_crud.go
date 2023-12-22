package crud

import (
	"errors"
	"fmt"
	"strings"

	"github.com/foliagecp/easyjson"
	"github.com/foliagecp/sdk/embedded/graph/common"
	sfplugins "github.com/foliagecp/sdk/statefun/plugins"
)

/*
	{
		"prefix": string, optional
		"body": json
	}

create types -> type link
*/
func CreateType(_ sfplugins.StatefunExecutor, contextProcessor *sfplugins.StatefunContextProcessor) {
	selfID := contextProcessor.Self.ID
	payload := contextProcessor.Payload

	prefix := payload.GetByPath("prefix").AsStringDefault("")

	_, err := contextProcessor.Request(sfplugins.GolangLocalRequest, "functions.graph.api.vertex.create", selfID, payload, nil)
	if err != nil {
		replyError(contextProcessor, err)
		return
	}

	link := easyjson.NewJSONObject()
	link.SetByPath("descendant_uuid", easyjson.NewJSON(selfID))
	link.SetByPath("link_type", easyjson.NewJSON("__type"))
	link.SetByPath("link_body.tags", easyjson.JSONFromArray([]string{"TYPE_" + selfID}))

	_, err = contextProcessor.Request(sfplugins.GolangLocalRequest, "functions.graph.api.link.create", prefix+"types", &link, nil)
	if err != nil {
		replyError(contextProcessor, err)
		return
	}

	replyOk(contextProcessor)
}

/*
	{
		"strategy": string, optional, default: DeepMerge
		"body": json
	}
*/
func UpdateType(_ sfplugins.StatefunExecutor, contextProcessor *sfplugins.StatefunContextProcessor) {
	selfID := contextProcessor.Self.ID

	payload := contextProcessor.Payload

	result, err := contextProcessor.Request(sfplugins.GolangLocalRequest, "functions.graph.api.vertex.update", selfID, payload, nil)
	if err := checkRequestError(result, err); err != nil {
		replyError(contextProcessor, err)
		return
	}

	replyOk(contextProcessor)
}

func DeleteType(_ sfplugins.StatefunExecutor, contextProcessor *sfplugins.StatefunContextProcessor) {
	replyOk(contextProcessor)
}

/*
	{
		"prefix": string, optional,
		"origin_type": string,
		"body": json
	}

create objects -> object link

create type -> object link

create object -> type link

TODO: Add origin type check
*/
func CreateObject(_ sfplugins.StatefunExecutor, contextProcessor *sfplugins.StatefunContextProcessor) {
	selfID := contextProcessor.Self.ID
	payload := contextProcessor.Payload

	originType, ok := payload.GetByPath("origin_type").AsString()
	if !ok {
		return
	}

	prefix := payload.GetByPath("prefix").AsStringDefault("")

	options := easyjson.NewJSONObjectWithKeyValue("return_op_stack", easyjson.NewJSON(true))
	result, err := contextProcessor.Request(sfplugins.GolangLocalRequest, "functions.graph.api.vertex.create", selfID, payload, &options)
	if err := checkRequestError(result, err); err != nil {
		replyError(contextProcessor, err)
		return
	}

	type _link struct {
		from, to, lt string
	}

	needLinks := []_link{
		{from: prefix + "objects", to: selfID, lt: "__object"},
		{from: selfID, to: prefix + originType, lt: "__type"},
		{from: prefix + originType, to: selfID, lt: "__object"},
	}

	for _, l := range needLinks {
		link := easyjson.NewJSONObject()
		link.SetByPath("descendant_uuid", easyjson.NewJSON(l.to))
		link.SetByPath("link_type", easyjson.NewJSON(l.lt))
		link.SetByPath("link_body", easyjson.NewJSONObject())

		switch l.lt {
		case "__type":
			link.SetByPath("link_body.tags", easyjson.JSONFromArray([]string{"TYPE_" + l.to}))
		}

		r, e := contextProcessor.Request(sfplugins.GolangLocalRequest, "functions.graph.api.link.create", l.from, &link, nil)
		if e := checkRequestError(r, e); e != nil {
			replyError(contextProcessor, e)
			return
		}
	}

	if result.PathExists("op_stack") {
		executeTriggersFromLLOpStack(contextProcessor, result.GetByPath("op_stack").GetPtr())
	}

	replyOk(contextProcessor)
}

/*
	{
		"mode": string, optional, default: merge
		"body": json
	}
*/
func UpdateObject(_ sfplugins.StatefunExecutor, contextProcessor *sfplugins.StatefunContextProcessor) {
	selfID := contextProcessor.Self.ID
	payload := contextProcessor.Payload

	options := easyjson.NewJSONObjectWithKeyValue("return_op_stack", easyjson.NewJSON(true))
	result, err := contextProcessor.Request(sfplugins.GolangLocalRequest, "functions.graph.api.vertex.update", selfID, payload, &options)
	if err := checkRequestError(result, err); err != nil {
		replyError(contextProcessor, err)
		return
	}

	if result.PathExists("op_stack") {
		executeTriggersFromLLOpStack(contextProcessor, result.GetByPath("op_stack").GetPtr())
	}

	replyOk(contextProcessor)
}

/*
	{
		"mode": "vertex" | "cascade", optional, default: vertex
	}
*/
func DeleteObject(_ sfplugins.StatefunExecutor, contextProcessor *sfplugins.StatefunContextProcessor) {
	selfID := contextProcessor.Self.ID
	payload := contextProcessor.Payload

	mode := payload.GetByPath("mode").AsStringDefault("vertex")
	switch mode {
	case "cascade":
		visited := map[string]struct{}{
			selfID: {},
		}
		queue := []string{selfID}

		for len(queue) > 0 {
			elem := queue[0]
			pattern := elem + ".out.ltp_oid-bdy.>"
			children := contextProcessor.GlobalCache.GetKeysByPattern(pattern)

			for _, v := range children {
				split := strings.Split(v, ".")
				if len(split) == 0 {
					continue
				}

				id := split[len(split)-1]

				if _, ok := visited[id]; ok {
					continue
				}

				visited[id] = struct{}{}
				queue = append(queue, id)
			}

			queue = queue[1:]
			if len(queue) == 0 {
				break
			}

			empty := easyjson.NewJSONObject()
			options := easyjson.NewJSONObjectWithKeyValue("return_op_stack", easyjson.NewJSON(true))
			result, err := contextProcessor.Request(sfplugins.GolangLocalRequest, "functions.graph.api.vertex.delete", elem, &empty, &options)
			if err := checkRequestError(result, err); err != nil {
				replyError(contextProcessor, err)
				return
			}

			if result.PathExists("op_stack") {
				executeTriggersFromLLOpStack(contextProcessor, result.GetByPath("op_stack").GetPtr())
			}
		}
	case "vertex":
	}

	replyOk(contextProcessor)
}

/*
	{
		"to": string,
		"object_link_type": string
		"body": json
	}

create type -> type link
*/
func CreateTypesLink(_ sfplugins.StatefunExecutor, contextProcessor *sfplugins.StatefunContextProcessor) {
	selfID := contextProcessor.Self.ID
	payload := contextProcessor.Payload

	objectLinkType, ok := payload.GetByPath("object_link_type").AsString()
	if !ok {
		return
	}

	to, ok := payload.GetByPath("to").AsString()
	if !ok {
		return
	}

	link := easyjson.NewJSONObject()
	link.SetByPath("descendant_uuid", easyjson.NewJSON(to))
	link.SetByPath("link_type", easyjson.NewJSON("__type"))
	link.SetByPath("link_body.link_type", easyjson.NewJSON(objectLinkType))
	link.SetByPath("link_body.tags", easyjson.JSONFromArray([]string{"TYPE_" + to}))

	result, err := contextProcessor.Request(sfplugins.GolangLocalRequest, "functions.graph.api.link.create", selfID, &link, nil)
	if err := checkRequestError(result, err); err != nil {
		replyError(contextProcessor, err)
		return
	}

	replyOk(contextProcessor)
}

/*
	{
		"to": string,
		"object_link_type": string, optional
		"body": json, optional
	}

if object_link_type not empty
  - prepare link_body.link_type = object_link_type
  - after success updating, find all objects with certain types and change link_type
*/
func UpdateTypesLink(_ sfplugins.StatefunExecutor, contextProcessor *sfplugins.StatefunContextProcessor) {
	selfID := contextProcessor.Self.ID
	payload := contextProcessor.Payload

	to, ok := payload.GetByPath("to").AsString()
	if !ok {
		replyError(contextProcessor, errors.New("to undefined"))
		return
	}

	objectLinkType := payload.GetByPath("object_link_type").AsStringDefault("")
	body := payload.GetByPath("body")

	if objectLinkType == "" && !body.IsNonEmptyObject() {
		replyError(contextProcessor, errors.New("nothing to update"))
		return
	}

	updateLinkPayload := easyjson.NewJSONObject()
	updateLinkPayload.SetByPath("descendant_uuid", easyjson.NewJSON(to))
	updateLinkPayload.SetByPath("link_type", easyjson.NewJSON("__type"))
	updateLinkPayload.SetByPath("link_body", body)
	updateLinkPayload.SetByPath("link_body.tags", easyjson.JSONFromArray([]string{"TYPE_" + to}))

	needUpdateObjectLinkType := objectLinkType != ""
	currentObjectLinkType := ""

	if needUpdateObjectLinkType {
		updateLinkPayload.SetByPath("link_body.link_type", easyjson.NewJSON(objectLinkType))

		currentBody, err := getTypesLinkBody(contextProcessor, selfID, to)
		if err != nil {
			replyError(contextProcessor, err)
			return
		}

		currentObjectLinkType, _ = currentBody.GetByPath("link_body.link_type").AsString()
	}

	result, err := contextProcessor.Request(sfplugins.GolangLocalRequest, "functions.graph.api.link.update", selfID, &updateLinkPayload, nil)
	if err := checkRequestError(result, err); err != nil {
		replyError(contextProcessor, err)
		return
	}

	// update link type of objects with certain types
	if needUpdateObjectLinkType {
		objects := findTypeObjects(contextProcessor, selfID)
		for _, objectID := range objects {
			pattern := fmt.Sprintf("%s.out.ltp_oid-bdy.%s.>", objectID, currentObjectLinkType)
			keys := contextProcessor.GlobalCache.GetKeysByPattern(pattern)

			for _, key := range keys {
				split := strings.Split(key, ".")
				toObjectID := split[len(split)-1]

				// update link_type
				updateObjectLinkPayload := easyjson.NewJSONObject()
				updateObjectLinkPayload.SetByPath("descendant_uuid", easyjson.NewJSON(toObjectID))
				updateObjectLinkPayload.SetByPath("link_type", easyjson.NewJSON(objectLinkType))
				updateObjectLinkPayload.SetByPath("link_body", easyjson.NewJSONObject())

				result, err := contextProcessor.Request(sfplugins.GolangLocalRequest, "functions.graph.api.link.update", objectID, &updateObjectLinkPayload, nil)
				if err := checkRequestError(result, err); err != nil {
					replyError(contextProcessor, err)
					return
				}
			}
		}
	}

	replyOk(contextProcessor)
}

func DeleteTypesLink(_ sfplugins.StatefunExecutor, contextProcessor *sfplugins.StatefunContextProcessor) {
	replyOk(contextProcessor)
}

/*
	{
		"to": string,
		"body": json
	}

create object -> object link
*/
func CreateObjectsLink(_ sfplugins.StatefunExecutor, contextProcessor *sfplugins.StatefunContextProcessor) {
	selfID := contextProcessor.Self.ID
	payload := contextProcessor.Payload

	objectToID, ok := payload.GetByPath("to").AsString()
	if !ok {
		replyError(contextProcessor, errors.New("to undefined"))
		return
	}

	linkType, err := getReferenceLinkTypeBetweenTwoObjects(contextProcessor, selfID, objectToID)
	if err != nil {
		replyError(contextProcessor, err)
		return
	}

	objectLink := easyjson.NewJSONObject()
	objectLink.SetByPath("descendant_uuid", easyjson.NewJSON(objectToID))
	objectLink.SetByPath("link_type", easyjson.NewJSON(linkType))
	objectLink.SetByPath("link_body", easyjson.NewJSONObject())

	options := easyjson.NewJSONObjectWithKeyValue("return_op_stack", easyjson.NewJSON(true))
	result, err := contextProcessor.Request(sfplugins.GolangLocalRequest, "functions.graph.api.link.create", selfID, &objectLink, &options)
	if err := checkRequestError(result, err); err != nil {
		replyError(contextProcessor, err)
		return
	}

	if result.PathExists("op_stack") {
		executeTriggersFromLLOpStack(contextProcessor, result.GetByPath("op_stack").GetPtr())
	}

	replyOk(contextProcessor)
}

/*
	{
		"to": string,
		"body": json
	}
*/
func UpdateObjectsLink(_ sfplugins.StatefunExecutor, contextProcessor *sfplugins.StatefunContextProcessor) {
	selfID := contextProcessor.Self.ID
	payload := contextProcessor.Payload

	objectToID, ok := payload.GetByPath("to").AsString()
	if !ok {
		return
	}

	linkType, err := getReferenceLinkTypeBetweenTwoObjects(contextProcessor, selfID, objectToID)
	if err != nil {
		replyError(contextProcessor, err)
		return
	}

	objectLink := easyjson.NewJSONObject()
	objectLink.SetByPath("descendant_uuid", easyjson.NewJSON(objectToID))
	objectLink.SetByPath("link_type", easyjson.NewJSON(linkType))
	objectLink.SetByPath("link_body", payload.GetByPath("body"))

	options := easyjson.NewJSONObjectWithKeyValue("return_op_stack", easyjson.NewJSON(true))
	result, err := contextProcessor.Request(sfplugins.GolangLocalRequest, "functions.graph.api.link.update", selfID, &objectLink, &options)
	if err := checkRequestError(result, err); err != nil {
		replyError(contextProcessor, err)
		return
	}

	if result.PathExists("op_stack") {
		executeTriggersFromLLOpStack(contextProcessor, result.GetByPath("op_stack").GetPtr())
	}

	replyOk(contextProcessor)
}

/*
	{
		"to": string,
	}
*/
func DeleteObjectsLink(_ sfplugins.StatefunExecutor, contextProcessor *sfplugins.StatefunContextProcessor) {
	selfID := contextProcessor.Self.ID
	payload := contextProcessor.Payload

	objectToID, ok := payload.GetByPath("to").AsString()
	if !ok {
		return
	}

	linkType, err := getReferenceLinkTypeBetweenTwoObjects(contextProcessor, selfID, objectToID)
	if err != nil {
		replyError(contextProcessor, err)
		return
	}

	objectLink := easyjson.NewJSONObject()
	objectLink.SetByPath("descendant_uuid", easyjson.NewJSON(objectToID))
	objectLink.SetByPath("link_type", easyjson.NewJSON(linkType))

	options := easyjson.NewJSONObjectWithKeyValue("return_op_stack", easyjson.NewJSON(true))
	result, err := contextProcessor.Request(sfplugins.GolangLocalRequest, "functions.graph.api.link.delete", selfID, &objectLink, &options)
	if err := checkRequestError(result, err); err != nil {
		replyError(contextProcessor, err)
		return
	}

	if result.PathExists("op_stack") {
		executeTriggersFromLLOpStack(contextProcessor, result.GetByPath("op_stack").GetPtr())
	}

	replyOk(contextProcessor)
}

func getReferenceLinkTypeBetweenTwoObjects(ctx *sfplugins.StatefunContextProcessor, fromObjectId, toObjectId string) (string, error) {
	fromTypeID := findObjectType(ctx, fromObjectId)
	toTypeID := findObjectType(ctx, toObjectId)

	linkBody, err := getTypesLinkBody(ctx, fromTypeID, toTypeID)
	if err != nil {
		return "", err
	}

	linkType, ok := linkBody.GetByPath("link_type").AsString()
	if !ok {
		return "", fmt.Errorf("type of a link was not defined in link type")
	}
	return linkType, nil
}

func executeTriggersFromLLOpStack(ctx *sfplugins.StatefunContextProcessor, opStack *easyjson.JSON) {
	if opStack != nil && opStack.IsArray() {
		for i := 0; i < opStack.ArraySize(); i++ {
			opData := opStack.ArrayElement(i)
			opStr := opData.GetByPath("op").AsStringDefault("")
			if len(opStr) > 0 {
				for j := 0; j < 3; j++ {
					if opStr == llAPIVertexCUDNames[j] {
						vId := opData.GetByPath("id").AsStringDefault("")
						if len(vId) > 0 && isVertexAnObject(ctx, vId) {
							var oldBody *easyjson.JSON = nil
							if opData.PathExists("old_body") {
								oldBody = opData.GetByPath("old_body").GetPtr()
							}
							var newBody *easyjson.JSON = nil
							if opData.PathExists("new_body") {
								newBody = opData.GetByPath("new_body").GetPtr()
							}
							executeObjectTriggers(ctx, vId, oldBody, newBody, j)
						}
					}
					if opStr == llAPILinkCUDNames[j] {
						fromVId := opData.GetByPath("from_id").AsStringDefault("")
						toVId := opData.GetByPath("to_id").AsStringDefault("")
						lType := opData.GetByPath("type").AsStringDefault("")
						if len(lType) > 0 && len(fromVId) > 0 && len(toVId) > 0 && isVertexAnObject(ctx, fromVId) && isVertexAnObject(ctx, toVId) {
							var oldBody *easyjson.JSON = nil
							if opData.PathExists("old_body") {
								oldBody = opData.GetByPath("old_body").GetPtr()
							}
							var newBody *easyjson.JSON = nil
							if opData.PathExists("new_body") {
								newBody = opData.GetByPath("new_body").GetPtr()
							}
							executeLinkTriggers(ctx, fromVId, toVId, lType, oldBody, newBody, j)
						}
					}
				}
			}
		}
	}
}

func isVertexAnObject(ctx *sfplugins.StatefunContextProcessor, id string) bool {
	return len(findObjectType(ctx, id)) > 0
}

func executeObjectTriggers(ctx *sfplugins.StatefunContextProcessor, objectID string, oldObjectBody, newObjectBody *easyjson.JSON, tt int /*0 - create, 1 - update, 2 - delete*/) {
	triggers := getObjectTypeTriggers(ctx, objectID)
	if triggers.IsNonEmptyObject() && tt > 0 && tt < 3 {
		elems := []string{"create", "update", "delete"}
		var functions []string
		if arr, ok := triggers.GetByPath(elems[tt]).AsArrayString(); ok {
			functions = arr
		}

		triggerData := easyjson.NewJSONObject()
		if tt > 0 {
			if oldObjectBody != nil {
				triggerData.SetByPath("old_body", *oldObjectBody)
			}
			if newObjectBody != nil {
				triggerData.SetByPath("new_body", *newObjectBody)
			}
		}
		payload := easyjson.NewJSONObject()
		payload.SetByPath(fmt.Sprintf("trigger.object.%s", elems[tt]), triggerData)

		for _, f := range functions {
			ctx.Signal(sfplugins.JetstreamGlobalSignal, f, objectID, &payload, nil)
		}
		// TODO: object deletion leads to object links deletions
	}
}

func executeLinkTriggers(ctx *sfplugins.StatefunContextProcessor, fromObjectId, toObjectId, linkType string, oldLinkBody, newLinkBody *easyjson.JSON, tt int /*0 - create, 1 - update, 2 - delete*/) {
	triggers := getObjectsLinkTypeTriggers(ctx, fromObjectId, toObjectId)
	if triggers.IsNonEmptyObject() && tt > 0 && tt < 3 {
		elems := []string{"create", "upadte", "delete"}
		var functions []string
		if arr, ok := triggers.GetByPath(elems[tt]).AsArrayString(); ok {
			functions = arr
		}

		referenceLinkType, err := getReferenceLinkTypeBetweenTwoObjects(ctx, fromObjectId, toObjectId)
		if err != nil || referenceLinkType != linkType {
			return
		}

		triggerData := easyjson.NewJSONObject()
		if tt > 0 {
			triggerData.SetByPath("to", easyjson.NewJSON(toObjectId))
			triggerData.SetByPath("type", easyjson.NewJSON(linkType))
			if oldLinkBody != nil {
				triggerData.SetByPath("old_body", *oldLinkBody)
			}
			if newLinkBody != nil {
				triggerData.SetByPath("new_body", *newLinkBody)
			}
		}
		payload := easyjson.NewJSONObject()
		payload.SetByPath(fmt.Sprintf("trigger.link.%s", elems[tt]), triggerData)

		for _, f := range functions {
			ctx.Signal(sfplugins.JetstreamGlobalSignal, f, fromObjectId, &payload, nil)
		}
	}
}

func getObjectTypeTriggers(ctx *sfplugins.StatefunContextProcessor, objectID string) easyjson.JSON {
	typeName := findObjectType(ctx, objectID)
	typeBody, err := ctx.GlobalCache.GetValueAsJSON(typeName)
	if err != nil || !typeBody.PathExists("triggers") {
		return easyjson.NewJSONObject()
	}
	return typeBody.GetByPath("triggers")
}

func getObjectsLinkTypeTriggers(ctx *sfplugins.StatefunContextProcessor, fromObjectId, toObjectId string) easyjson.JSON {
	fromTypeName := findObjectType(ctx, fromObjectId)
	toTypeName := findObjectType(ctx, toObjectId)
	typesLinkBody, err := getTypesLinkBody(ctx, fromTypeName, toTypeName)
	if err != nil || !typesLinkBody.PathExists("triggers") {
		return easyjson.NewJSONObject()
	}
	return typesLinkBody.GetByPath("triggers")
}

func findObjectType(ctx *sfplugins.StatefunContextProcessor, objectID string) string {
	pattern := objectID + ".out.ltp_oid-bdy.__type.>"

	keys := ctx.GlobalCache.GetKeysByPattern(pattern)
	if len(keys) == 0 {
		return ""
	}

	split := strings.Split(keys[0], ".")

	return split[len(split)-1]
}

func findTypeObjects(ctx *sfplugins.StatefunContextProcessor, typeID string) []string {
	pattern := typeID + ".out.ltp_oid-bdy.__object.>"

	keys := ctx.GlobalCache.GetKeysByPattern(pattern)
	if len(keys) == 0 {
		return []string{}
	}

	out := make([]string, 0, len(keys))
	for _, v := range keys {
		split := strings.Split(v, ".")
		out = append(out, split[len(split)-1])
	}

	return out
}

func getTypesLinkBody(ctx *sfplugins.StatefunContextProcessor, from, to string) (*easyjson.JSON, error) {
	id := fmt.Sprintf("%s.out.ltp_oid-bdy.__type.%s", from, to)

	body, err := ctx.GlobalCache.GetValueAsJSON(id)
	if err != nil {
		return nil, fmt.Errorf("link %s, %s not found: %w", from, to, err)
	}

	return body, nil
}

func replyOk(ctx *sfplugins.StatefunContextProcessor, msg ...string) {
	reply(ctx, "ok", msg)
}

func replyError(ctx *sfplugins.StatefunContextProcessor, err error) {
	reply(ctx, "failed", err.Error())
}

func reply(ctx *sfplugins.StatefunContextProcessor, status string, data any) {
	qid := common.GetQueryID(ctx)
	reply := easyjson.NewJSONObject()
	reply.SetByPath("status", easyjson.NewJSON(status))
	reply.SetByPath("result", easyjson.NewJSON(data))
	common.ReplyQueryID(qid, easyjson.NewJSONObjectWithKeyValue("payload", reply).GetPtr(), ctx)
}

func checkRequestError(result *easyjson.JSON, err error) error {
	if err != nil {
		return err
	}

	if result.GetByPath("status").AsStringDefault("failed") == "failed" {
		return errors.New(result.GetByPath("result").AsStringDefault("unknown error"))
	}

	return nil
}
