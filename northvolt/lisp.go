package northvolt

import (
	"context"
	"fmt"

	"github.com/deosjr/whistle/lisp"
	"github.com/northvolt/go-service-digitaltwin/digitaltwin"
	"github.com/northvolt/go-service-digitaltwin/digitaltwin/digitaltwinhttp"
	"github.com/northvolt/go-service-process/process"
	"github.com/northvolt/go-service-process/process/processhttp"
	"github.com/northvolt/go-service/localrunner"
	"github.com/northvolt/graphql-schema/model"
)

// caching northvolt api calls so we dont overwhelm it
// pages run their script _each frame_ right now
// map from apiendpoint to input to data
var cache = map[string]map[lisp.SExpression]lisp.SExpression{}

var (
	dt digitaltwin.Service
	ps process.Service
)

// wrapper around nv service calls

func Load(env *lisp.Env) {
	r := localrunner.NewLocalRunner()
	dt = digitaltwinhttp.NewClient(r.FixedInstancer("/digitaltwin")).WithReqModifier(r.AuthorizeHeader())
	ps = processhttp.NewClient(r.FixedInstancer("/process")).WithReqModifier(r.AuthorizeHeader())
	cache["dt:identity"] = map[lisp.SExpression]lisp.SExpression{}
	cache["ps:results"] = map[lisp.SExpression]lisp.SExpression{}

	// digitaltwin
	env.AddBuiltin("dt:identity", dtIdentity)
	env.AddBuiltin("identity->cell", identity2cell)
	env.AddBuiltin("cell:id", cellID)

	// process
	env.AddBuiltin("ps:results", psResults)
	env.AddBuiltin("pr:kind", presultKind)
}

func dtIdentity(args []lisp.SExpression) (lisp.SExpression, error) {
	ctx := context.Background()
	arg0 := args[0]
	prev, ok := cache["dt:identity"][arg0]
	if ok {
		return prev, nil
	}

	nvid := arg0.AsPrimitive().(string)
	fmt.Printf("calling digitaltwin identity %s\n", nvid)
	identity, err := dt.Identity(ctx, nvid)
	if err != nil {
		return nil, err
	}
	result := lisp.NewPrimitive(identity)
	cache["dt:identity"][arg0] = result
	return result, nil
}

func identity2cell(args []lisp.SExpression) (lisp.SExpression, error) {
	identity := args[0].AsPrimitive().(model.NorthvoltIdentity)
	cell, ok := identity.(*model.Cell)
	if !ok {
		return nil, fmt.Errorf("not a cell!")
	}
	return lisp.NewPrimitive(cell), nil
}

func cellID(args []lisp.SExpression) (lisp.SExpression, error) {
	cell := args[0].AsPrimitive().(*model.Cell)
	return lisp.NewPrimitive(cell.GetID()), nil
}

func psResults(args []lisp.SExpression) (lisp.SExpression, error) {
	ctx := context.Background()
	arg0 := args[0]
	prev, ok := cache["ps:results"][arg0]
	if ok {
		return prev, nil
	}

	nvid := arg0.AsPrimitive().(string)
	fmt.Printf("calling process results %s\n", nvid)
	results, err := ps.Results(ctx, nvid)
	if err != nil {
		return nil, err
	}
	sexprs := make([]lisp.SExpression, len(results))
	for i, r := range results {
		sexprs[i] = lisp.NewPrimitive(r)
	}
	result := lisp.MakeConsList(sexprs)
	cache["ps:results"][arg0] = result
	return result, nil
}

func presultKind(args []lisp.SExpression) (lisp.SExpression, error) {
	pr := args[0].AsPrimitive().(*model.DefaultProcessResult)
	return lisp.NewPrimitive(string(pr.Kind)), nil
}
