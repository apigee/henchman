package henchman

import (
	"github.com/flosch/pongo2"
)

func init() {
	pongo2.RegisterFilter("ok", filterOk)
	pongo2.RegisterFilter("changed", filterChanged)
	pongo2.RegisterFilter("skipped", filterSkipped)
	pongo2.RegisterFilter("failure", filterFailure)
	pongo2.RegisterFilter("error", filterError)
	pongo2.RegisterFilter("unreachable", filterUnreachable)
}

func filterOk(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	return pongo2.AsValue(in.String() == "ok"), nil
}

func filterChanged(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	return pongo2.AsValue(in.String() == "changed"), nil
}

func filterSkipped(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	return pongo2.AsValue(in.String() == "skipped"), nil
}

func filterFailure(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	return pongo2.AsValue(in.String() == "failure"), nil
}

func filterError(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	return pongo2.AsValue(in.String() == "error"), nil
}

func filterUnreachable(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	return pongo2.AsValue(in.String() == "unreachable"), nil
}
