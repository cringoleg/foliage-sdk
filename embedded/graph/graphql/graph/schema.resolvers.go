package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.
// Code generated by github.com/99designs/gqlgen version v0.17.48

import (
	"context"
	"fmt"

	"github.com/foliagecp/easyjson"
	"github.com/foliagecp/sdk/embedded/graph/graphql/extra"
	"github.com/foliagecp/sdk/embedded/graph/graphql/graph/model"
	sfMediators "github.com/foliagecp/sdk/statefun/mediator"
	sfPlugins "github.com/foliagecp/sdk/statefun/plugins"
)

// SearchObjects is the resolver for the searchObjects field.
func (r *queryResolver) SearchObjects(ctx context.Context, query string, objectTypes []string, requestFields []string) ([]*model.Object, error) {
	result := []*model.Object{}

	if DBC != nil {
		payload := easyjson.NewJSONObjectWithKeyValue("query", easyjson.NewJSON(query))
		payload.SetByPath("object_type_filter", easyjson.JSONFromArray(objectTypes))
		msg := sfMediators.OpMsgFromSfReply(DBC.Request(sfPlugins.AutoRequestSelect, "functions.graph.api.search.objects.fvpm", "root", &payload, nil))

		if msg.Status != sfMediators.SYNC_OP_STATUS_OK {
			return result, fmt.Errorf("error requesting foliage search function, status %d: %s", msg.Status, msg.Details)
		}

		matchObjects := msg.Data.GetByPath("match.objects")
		if len(requestFields) == 0 {
			if a, ok := msg.Data.GetByPath("match.fields").AsArrayString(); ok {
				requestFields = a
			}
		}

		for _, objectId := range matchObjects.ObjectKeys() {
			objectData := matchObjects.GetByPath(objectId)
			otype := objectData.GetByPath("type").AsStringDefault("")
			body := objectData.GetByPath("body")

			resObject := model.Object{ID: objectId, Type: otype, RequestFields: extra.JSON{}}
			for _, f := range requestFields {
				v := body.GetByPath(f)
				if body.PathExists(f) && (v.IsString() || v.IsBool() || v.IsNumeric()) {
					resObject.RequestFields[f] = body.GetByPath(f).Value
				} else {
					resObject.RequestFields[f] = nil
				}
			}
			result = append(result, &resObject)
		}
		return result, nil
	}
	return result, fmt.Errorf("core's api client is invalid")
}

// Query returns QueryResolver implementation.
func (r *Resolver) Query() QueryResolver { return &queryResolver{r} }

type queryResolver struct{ *Resolver }
